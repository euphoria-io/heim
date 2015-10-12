package jobs

import "fmt"

var (
	ErrInvalidJobType        = fmt.Errorf("invalid job type")
	ErrJobCancelled          = fmt.Errorf("job cancelled")
	ErrJobCanceled           = ErrJobCancelled
	ErrJobNotFound           = fmt.Errorf("job not found")
	ErrJobNotClaimed         = fmt.Errorf("job not claimed")
	ErrJobQueueAlreadyExists = fmt.Errorf("job queue already exists")
	ErrJobQueueNotFound      = fmt.Errorf("job queue not found")
)
