package jobs

import (
	"bytes"
	"math/rand"
	"time"

	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type JobClaim struct {
	bytes.Buffer
	JobID         snowflake.Snowflake
	HandlerID     string
	AttemptNumber int32
	Queue         JobQueue
}

func (jc *JobClaim) Fail(ctx scope.Context, reason string) error {
	if jc == nil {
		return ErrJobNotClaimed
	}
	return jc.Queue.Fail(ctx, jc.JobID, jc.HandlerID, jc.AttemptNumber, reason, jc.Bytes())
}

func (jc *JobClaim) Complete(ctx scope.Context) error {
	if jc == nil {
		return ErrJobNotClaimed
	}
	return jc.Queue.Complete(ctx, jc.JobID, jc.HandlerID, jc.AttemptNumber, jc.Bytes())
}

func Claim(ctx scope.Context, jq JobQueue, handlerID string, pollTime time.Duration, stealChance float64) (*Job, error) {
	for ctx.Err() == nil {
		if rand.Float64() < stealChance {
			job, err := jq.TrySteal(ctx, handlerID)
			if err != nil && err != ErrJobNotFound {
				return nil, err
			}
			if err == nil {
				return job, nil
			}
		}
		job, err := jq.TryClaim(ctx, handlerID)
		if err != nil {
			if err == ErrJobNotFound {
				child := ctx.ForkWithTimeout(pollTime)
				if err = jq.WaitForJob(child); err != nil && err != scope.TimedOut {
					return nil, err
				}
				continue
			}
			return nil, err
		}
		return job, nil
	}
	return nil, ctx.Err()
}
