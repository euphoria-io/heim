package jobs

import (
	"math/rand"
	"time"

	"euphoria.io/scope"
)

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
