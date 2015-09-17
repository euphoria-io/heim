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

func (jq *JobQueue) Add(
	ctx scope.Context, jobType jobs.JobType, payload interface{}, options ...jobs.JobOption) (
	snowflake.Snowflake, error) {

	jobID, err := snowflake.New()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	e := entry{
		Job: jobs.Job{
			ID:                jobID,
			Type:              jobType,
			Created:           now,
			Due:               now,
			AttemptsRemaining: math.MaxInt32,
			MaxWorkDuration:   jobs.DefaultMaxWorkDuration,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	if err := e.Data.UnmarshalJSON(data); err != nil {
		return 0, err
	}

	for _, option := range options {
		if err := option.Apply(&e.Job); err != nil {
			return 0, err
		}
	}

	heap.Push(&jq.available, e)

	jq.m.Lock()
	if jq.c != nil {
		jq.c.Signal()
	}
	jq.m.Unlock()

	return jobID, nil
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

func (jq *JobQueue) Claim(ctx scope.Context, handlerID string) (jobs.Job, error) {
	child := ctx.Fork()
	ch := make(chan *jobs.Job)

	// polling goroutine, scheduled by condition
	go func() {
		// send an initial nil value to inform caller that we're ready
		ch <- nil

		// wait for caller to respond
		// caller will lock mutex for us and wait for us to unlock it
		<-ch
		defer jq.m.Unlock()

		// loop until we claim a job or get cancelled
		for child.Err() == nil {
			if len(jq.available) > 0 {
				e := heap.Pop(&jq.available).(entry)
				e.claimed = time.Now()
				e.AttemptsRemaining -= 1
				e.JobClaim = &jobs.JobClaim{
					JobID:         e.ID,
					HandlerID:     handlerID,
					AttemptNumber: e.AttemptsMade + 1,
					Queue:         jq,
				}
				heap.Push(&jq.working, e)
				ch <- &e.Job
				return
			}
			jq.c.Wait()
		}
	}()

	// to facilitate testing, wait for initial nil value from polling goroutine
	// before coordinating with breakpoint
	<-ch
	if err := ctx.Check("euphoria.io/heim/proto/jobs.JobQueue.Claim"); err != nil {
		child.Terminate(err)
	}
	jq.m.Lock()
	ch <- nil

	select {
	case <-child.Done():
		jq.m.Lock()
		jq.c.Broadcast()
		// job may still have been received between receiving cancellation signal
		// and locking, so return it to the queue without penalty
		if j, ok := <-ch; ok {
			jq.release(j.ID, 0)
		}
		jq.m.Unlock()
		return jobs.Job{}, child.Err()
	case job := <-ch:
		return *job, nil
	}
}

func (jq *JobQueue) Steal(ctx scope.Context, handlerID string) (jobs.Job, error) {
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
		return jobs.Job{}, jobs.ErrJobNotFound
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
			return entry.Job, nil
		}
	}

	return jobs.Job{}, fmt.Errorf("job disappeared")
}
