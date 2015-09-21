package mock

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"euphoria.io/heim/proto/jobs"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type JobService struct {
	m  sync.Mutex
	qs map[string]*JobQueue
}

func (js *JobService) CreateQueue(ctx scope.Context, name string) (jobs.JobQueue, error) {
	js.m.Lock()
	defer js.m.Unlock()

	if js.qs == nil {
		js.qs = map[string]*JobQueue{}
	}
	if _, ok := js.qs[name]; ok {
		return nil, jobs.ErrJobQueueAlreadyExists
	}

	jq := &JobQueue{}
	jq.c = sync.NewCond(&jq.m)
	js.qs[name] = jq
	return jq, nil
}

func (js *JobService) GetQueue(ctx scope.Context, name string) (jobs.JobQueue, error) {
	jq, ok := js.qs[name]
	if !ok {
		return nil, jobs.ErrJobQueueNotFound
	}
	return jq, nil
}

type entry struct {
	jobs.Job
	claimed time.Time
}

type jobHeap []entry

func (es jobHeap) Len() int            { return len(es) }
func (es jobHeap) Less(i, j int) bool  { return es[i].Due.Before(es[j].Due) }
func (es jobHeap) Swap(i, j int)       { es[j], es[i] = es[i], es[j] }
func (es *jobHeap) Push(x interface{}) { *es = append(*es, x.(entry)) }

func (es *jobHeap) Pop() interface{} {
	final := (*es)[len(*es)-1]
	*es = (*es)[:len(*es)-1]
	return final
}

type JobQueue struct {
	m         sync.Mutex
	c         *sync.Cond
	available jobHeap
	working   jobHeap
}

func (jq *JobQueue) newJob(jobType jobs.JobType, payload interface{}, options ...jobs.JobOption) (*jobs.Job, error) {
	jobID, err := snowflake.New()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	job := &jobs.Job{
		ID:                jobID,
		Type:              jobType,
		Created:           now,
		Due:               now,
		AttemptsRemaining: math.MaxInt32,
		MaxWorkDuration:   jobs.DefaultMaxWorkDuration,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if err := job.Data.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	for _, option := range options {
		if err := option.Apply(job); err != nil {
			return nil, err
		}
	}

	return job, nil
}

func (jq *JobQueue) Add(
	ctx scope.Context, jobType jobs.JobType, payload interface{}, options ...jobs.JobOption) (
	snowflake.Snowflake, error) {

	job, err := jq.newJob(jobType, payload, options...)
	if err != nil {
		return 0, err
	}

	jq.m.Lock()
	heap.Push(&jq.available, entry{Job: *job})
	if jq.c != nil {
		jq.c.Signal()
	}
	jq.m.Unlock()

	return job.ID, nil
}

func (jq *JobQueue) AddAndClaim(
	ctx scope.Context, jobType jobs.JobType, payload interface{}, handlerID string, options ...jobs.JobOption) (
	*jobs.Job, error) {

	job, err := jq.newJob(jobType, payload, options...)
	if err != nil {
		return nil, err
	}

	job.AttemptsMade = 1
	job.AttemptsRemaining -= 1
	job.JobClaim = &jobs.JobClaim{
		JobID:         job.ID,
		HandlerID:     handlerID,
		AttemptNumber: 0,
		Queue:         jq,
	}
	e := entry{
		Job:     *job,
		claimed: job.Created,
	}

	jq.m.Lock()
	heap.Push(&jq.working, e)
	jq.m.Unlock()

	return job, nil
}

func (jq *JobQueue) Complete(ctx scope.Context, jobID snowflake.Snowflake, handlerID string, attempt int32, log []byte) error {
	jq.m.Lock()
	defer jq.m.Unlock()

	return jq.remove(jobID, nil)
}

func (jq *JobQueue) Fail(ctx scope.Context, jobID snowflake.Snowflake, handlerID string, attempt int32, reason string, log []byte) error {
	jq.m.Lock()
	defer jq.m.Unlock()

	jq.release(jobID, 0)
	jq.c.Signal()
	return nil
}

func (jq *JobQueue) Cancel(ctx scope.Context, jobID snowflake.Snowflake) error {
	jq.m.Lock()
	defer jq.m.Unlock()

	return jq.remove(jobID, jobs.ErrJobCancelled)
}

func (jq *JobQueue) release(jobID snowflake.Snowflake, penalty int32) {
	for i, entry := range jq.working {
		if entry.ID == jobID {
			heap.Remove(&jq.working, i)
			entry.AttemptsRemaining += penalty
			entry.AttemptsMade = entry.JobClaim.AttemptNumber
			entry.JobClaim = nil
			heap.Push(&jq.available, entry)
			return
		}
	}
}

func (jq *JobQueue) remove(jobID snowflake.Snowflake, err error) error {
	for i, entry := range jq.working {
		if entry.ID == jobID {
			heap.Remove(&jq.working, i)
			return nil
		}
	}

	for i, entry := range jq.available {
		if entry.ID == jobID {
			heap.Remove(&jq.available, i)
			return nil
		}
	}

	return jobs.ErrJobNotFound
}

func (jq *JobQueue) Stats(ctx scope.Context) (jobs.JobQueueStats, error) {
	jq.m.Lock()
	defer jq.m.Unlock()

	now := time.Now()
	stats := jobs.JobQueueStats{}
	for _, entry := range jq.available {
		stats.Waiting++
		if !now.Before(entry.Due) {
			stats.Due++
		}
	}
	for _, entry := range jq.working {
		if now.Before(entry.claimed.Add(entry.MaxWorkDuration)) {
			stats.Claimed++
		} else {
			stats.Waiting++
		}
		if !now.Before(entry.Due) {
			stats.Due++
		}
	}
	return stats, nil
}

func (jq *JobQueue) WaitForJob(ctx scope.Context) error {
	ch := make(chan error)

	go func() {
		jq.m.Lock()
		jq.c.Wait()
		jq.m.Unlock()
		ch <- nil
	}()

	select {
	case <-ctx.Done():
		jq.m.Lock()
		jq.c.Broadcast()
		jq.m.Unlock()
		<-ch
		return ctx.Err()
	case err := <-ch:
		return err
	}
}

func (jq *JobQueue) TryClaim(ctx scope.Context, handlerID string) (*jobs.Job, error) {
	jq.m.Lock()
	defer jq.m.Unlock()

	if len(jq.available) == 0 {
		return nil, jobs.ErrJobNotFound
	}

	e := heap.Pop(&jq.available).(entry)
	e.AttemptsRemaining -= 1
	e.JobClaim = &jobs.JobClaim{
		JobID:         e.ID,
		HandlerID:     handlerID,
		AttemptNumber: e.AttemptsMade + 1,
		Queue:         jq,
	}
	ret := &jobs.Job{}
	*ret = e.Job
	e.claimed = time.Now()
	e.AttemptsMade += 1
	heap.Push(&jq.working, e)
	return &e.Job, nil
}

func (jq *JobQueue) TrySteal(ctx scope.Context, handlerID string) (*jobs.Job, error) {
	jq.m.Lock()
	defer jq.m.Unlock()

	var maxOverrun time.Duration
	idx := -1
	now := time.Now()
	for i, entry := range jq.working {
		if entry.JobClaim.HandlerID == handlerID {
			continue
		}
		overrun := now.Sub(entry.claimed) - entry.MaxWorkDuration
		if overrun < 0 {
			continue
		}
		if idx < 0 || overrun > maxOverrun {
			idx = i
			maxOverrun = overrun
		}
	}
	if idx < 0 {
		return nil, jobs.ErrJobNotFound
	}

	jobID := jq.working[idx].ID
	jq.release(jobID, 0)
	for i, entry := range jq.available {
		if entry.ID == jobID {
			heap.Remove(&jq.available, i)
			entry.claimed = now
			entry.AttemptsMade += 1
			entry.AttemptsRemaining -= 1
			entry.JobClaim = &jobs.JobClaim{
				JobID:         entry.ID,
				HandlerID:     handlerID,
				AttemptNumber: entry.AttemptsMade + 1,
				Queue:         jq,
			}
			heap.Push(&jq.working, entry)
			return &entry.Job, nil
		}
	}

	return nil, fmt.Errorf("job disappeared")
}
