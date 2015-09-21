package jobs

import (
	"time"

	"euphoria.io/scope"
)

func Claim(ctx scope.Context, jq JobQueue, handlerID string, pollTime time.Duration) (*Job, error) {
	for ctx.Err() == nil {
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
