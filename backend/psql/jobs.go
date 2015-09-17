package psql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
	"gopkg.in/gorp.v1"
)

type JobQueue struct {
	Name string
}

func (jq *JobQueue) Bind(b *Backend) *JobQueueBinding {
	return &JobQueueBinding{
		JobQueue: jq,
		Backend:  b,
	}
}

type JobQueueBinding struct {
	*JobQueue
	*Backend

	m sync.Mutex
	c *sync.Cond
}

type JobItem struct {
	ID                     int64
	Queue                  string
	JobType                string `db:"job_type"`
	Data                   []byte
	Created                time.Time
	Due                    time.Time
	Claimed                gorp.NullTime
	Completed              gorp.NullTime
	MaxWorkDurationSeconds int32 `db:"max_work_duration_seconds"`
	AttemptsMade           int32 `db:"attempts_made"`
	AttemptsRemaining      int32 `db:"attempts_remaining"`
}

type JobLog struct {
	JobID     int64 `db:"job_id"`
	Attempt   int32
	HandlerID string `db:"handler_id"`
	Started   time.Time
	Finished  gorp.NullTime
	Stolen    gorp.NullTime
	StolenBy  sql.NullString `db:"stolen_by"`
	Outcome   sql.NullString
	Log       []byte
}

type JobService struct {
	*Backend
}

func (js *JobService) CreateQueue(ctx scope.Context, name string) (proto.JobQueue, error) {
	jq := &JobQueue{Name: name}
	if err := js.DbMap.Insert(jq); err != nil {
		if strings.HasPrefix(err.Error(), "pq: duplicate key value") {
			return nil, proto.ErrJobQueueAlreadyExists
		}
		return nil, err
	}
	return jq.Bind(js.Backend), nil
}

func (js *JobService) GetQueue(ctx scope.Context, name string) (proto.JobQueue, error) {
	row, err := js.DbMap.Get(JobQueue{}, name)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, proto.ErrJobQueueNotFound
	}
	return row.(*JobQueue).Bind(js.Backend), nil
}

func (jq *JobQueueBinding) Add(
	ctx scope.Context, jobType proto.JobType, payload interface{}, options ...proto.JobOption) (
	snowflake.Snowflake, error) {

	jobID, err := snowflake.New()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	job := &proto.Job{
		ID:                jobID,
		Type:              jobType,
		Created:           now,
		Due:               now,
		AttemptsRemaining: math.MaxInt32,
		MaxWorkDuration:   proto.DefaultMaxWorkDuration,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	if err := job.Data.UnmarshalJSON(data); err != nil {
		return 0, err
	}

	for _, option := range options {
		if err := option.Apply(job); err != nil {
			return 0, err
		}
	}

	item := &JobItem{
		ID:                     int64(jobID),
		Queue:                  jq.Name,
		JobType:                string(jobType),
		Data:                   []byte(job.Data),
		Created:                job.Created,
		Due:                    job.Due,
		AttemptsRemaining:      job.AttemptsRemaining,
		MaxWorkDurationSeconds: int32(job.MaxWorkDuration / time.Second),
	}

	t, err := jq.DbMap.Begin()
	if err != nil {
		return 0, err
	}

	if err := t.Insert(item); err != nil {
		rollback(ctx, t)
		return 0, err
	}

	escaped := strings.Replace(jq.Name, "'", "''", -1)
	if _, err := t.Exec(fmt.Sprintf("NOTIFY job_item, '%s'", escaped)); err != nil {
		rollback(ctx, t)
		return 0, err
	}

	if err := t.Commit(); err != nil {
		return 0, err
	}

	return jobID, nil
}

func (jq *JobQueueBinding) Claim(ctx scope.Context, handlerID string) (proto.Job, error) {
	cols, err := allColumns(jq.Backend.DbMap, JobItem{}, "")
	if err != nil {
		return proto.Job{}, err
	}

	jql := jq.Backend.jobQueueListener()
	child := ctx.Fork()
	ch := make(chan *proto.Job, 0)

	// polling goroutine, scheduled by backend-managed condition
	var pollErr error
	go func() {

		// send an initial nil value to inform caller that we're waiting
		ch <- nil

		// wait for caller to allow us to proceed
		<-ch

		// loop until we claim a job or get cancelled
		for child.Err() == nil {
			var row JobItem
			err := jq.Backend.DbMap.SelectOne(&row, fmt.Sprintf("SELECT %s FROM job_claim($1, $2)", cols), jq.Name, handlerID)
			if err != nil {
				if err == sql.ErrNoRows {
					// no job was ready; wait for notification of new job and try again
					jql.wait(jq.Name)
					continue
				}
				pollErr = err
				ch <- nil
				return
			}
			job := &proto.Job{
				ID:                snowflake.Snowflake(row.ID),
				Type:              proto.JobType(row.JobType),
				Data:              json.RawMessage(row.Data),
				Created:           row.Created,
				Due:               row.Due,
				MaxWorkDuration:   time.Duration(row.MaxWorkDurationSeconds) * time.Second,
				AttemptsMade:      row.AttemptsMade,
				AttemptsRemaining: row.AttemptsRemaining - 1,
				JobClaim: &proto.JobClaim{
					JobID:         snowflake.Snowflake(row.ID),
					HandlerID:     handlerID,
					AttemptNumber: row.AttemptsMade + 1,
					Queue:         jq,
				},
			}
			ch <- job
			return
		}
	}()

	// to facilitate testing, wait for initial nil value from polling goroutine
	// before coordinating with breakpoint
	<-ch
	if err := ctx.Check("euphoria.io/heim/proto.JobQueue.Claim"); err != nil {
		child.Terminate(err)
		ch <- nil
	}
	ch <- nil

	select {
	case <-child.Done():
		jql.wakeAll(jq.Name)
		// job may still have been received between receiving cancellation signal
		// and locking
		if j, ok := <-ch; ok && j != nil {
			// TODO: release without penalty instead of returning?
			return *j, nil
		}
		return proto.Job{}, child.Err()
	case job := <-ch:
		if job == nil {
			return proto.Job{}, pollErr
		}
		return *job, nil
	}
}

func (jq *JobQueueBinding) Steal(ctx scope.Context, handlerID string) (proto.Job, error) {
	var row JobItem
	cols, err := allColumns(jq.Backend.DbMap, JobItem{}, "")
	if err != nil {
		return proto.Job{}, err
	}
	err = jq.Backend.DbMap.SelectOne(&row, fmt.Sprintf("SELECT %s FROM job_steal($1, $2)", cols), jq.Name, handlerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return proto.Job{}, proto.ErrJobNotFound
		}
		return proto.Job{}, err
	}
	job := proto.Job{
		ID:                snowflake.Snowflake(row.ID),
		Type:              proto.JobType(row.JobType),
		Data:              json.RawMessage(row.Data),
		Created:           row.Created,
		Due:               row.Due,
		MaxWorkDuration:   time.Duration(row.MaxWorkDurationSeconds) * time.Second,
		AttemptsMade:      row.AttemptsMade,
		AttemptsRemaining: row.AttemptsRemaining - 1,
		JobClaim: &proto.JobClaim{
			JobID:         snowflake.Snowflake(row.ID),
			HandlerID:     handlerID,
			AttemptNumber: row.AttemptsMade + 1,
			Queue:         jq,
		},
	}
	return job, nil
}

func (jq *JobQueueBinding) Cancel(ctx scope.Context, jobID snowflake.Snowflake) error {
	_, err := jq.Backend.DbMap.Exec("SELECT job_cancel($1)", jobID)
	return err
}

func (jq *JobQueueBinding) Complete(
	ctx scope.Context, jobID snowflake.Snowflake, handlerID string, attemptNumber int32, log []byte) error {
	_, err := jq.Backend.DbMap.Exec("SELECT job_complete($1,$2,$3)", jobID, attemptNumber, log)
	return err
}

func (jq *JobQueueBinding) Fail(
	ctx scope.Context, jobID snowflake.Snowflake, handlerID string, attemptNumber int32, reason string, log []byte) error {

	_, err := jq.Backend.DbMap.Exec("SELECT job_fail($1,$2,$3,$4)", jobID, attemptNumber, reason, log)
	return err
}

func (jq *JobQueueBinding) Stats(ctx scope.Context) (proto.JobQueueStats, error) {
	var stats proto.JobQueueStats

	err := jq.Backend.DbMap.SelectOne(
		&stats,
		"SELECT COUNT(*)-SUM(is_claimed) AS waiting, SUM(is_due) AS due, SUM(is_claimed) AS claimed FROM ("+
			"SELECT CASE WHEN due <= NOW() THEN 1 ELSE 0 END AS is_due,"+
			" CASE WHEN jl.job_id IS NOT NULL AND jl.started + job.max_work_duration_seconds * interval '1 second' > NOW() THEN 1 ELSE 0 END AS is_claimed"+
			" FROM job_item job LEFT JOIN job_log jl ON job.id = jl.job_id AND jl.attempt = job.attempts_made-1"+
			" WHERE job.queue = $1 AND job.completed IS NULL) AS t1",
		jq.Name)
	return stats, err
}

func newJobQueueListener(b *Backend) *jobQueueListener {
	jql := &jobQueueListener{
		Backend: b,
	}
	b.ctx.WaitGroup().Add(1)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go jql.background(wg)
	wg.Wait()
	return jql
}

type jobQueueListener struct {
	Backend *Backend
	m       sync.Mutex
	cs      map[string]*sync.Cond
}

func (jql *jobQueueListener) background(wg *sync.WaitGroup) {
	ctx := jql.Backend.ctx.Fork()
	logger := jql.Backend.logger

	defer ctx.WaitGroup().Done()

	listener := pq.NewListener(jql.Backend.dsn, 200*time.Millisecond, 5*time.Second, nil)
	if err := listener.Listen("job_item"); err != nil {
		// TODO: manage this more nicely
		panic("job listen: " + err.Error())
	}
	logger.Printf("job listener started")

	// Signal to constructor that we're ready to handle operations.
	wg.Done()

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			// Ping to make sure the database connection is still live.
			if err := listener.Ping(); err != nil {
				logger.Printf("job listener ping: %s\n", err)
				jql.Backend.ctx.Terminate(fmt.Errorf("job listener ping: %s", err))
				return
			}
		case notice := <-listener.Notify:
			if notice == nil {
				logger.Printf("job listener: received nil notification")
				// A nil notice indicates a loss of connection.
				// For now it's easier to just shut down and force job
				// processor to restart.
				jql.Backend.ctx.Terminate(ErrPsqlConnectionLost)
				return
			}

			jql.m.Lock()
			if c, ok := jql.cs[notice.Extra]; ok {
				c.Signal()
			}
			jql.m.Unlock()
		}
	}
}

func (jql *jobQueueListener) wait(queueName string) {
	jql.m.Lock()
	defer jql.m.Unlock()

	if jql.cs == nil {
		jql.cs = map[string]*sync.Cond{}
	}
	c, ok := jql.cs[queueName]
	if !ok {
		c = sync.NewCond(&jql.m)
		jql.cs[queueName] = c
	}
	c.Wait()
}

func (jql *jobQueueListener) wakeAll(queueName string) {
	jql.m.Lock()
	defer jql.m.Unlock()

	if c, ok := jql.cs[queueName]; ok {
		c.Broadcast()
	}
}
