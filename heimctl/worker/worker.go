package worker

import (
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/jobs"
	"euphoria.io/scope"
)

type Worker interface {
	Init(heim *proto.Heim) error
	QueueName() string
	JobType() jobs.JobType
	Work(ctx scope.Context, job *jobs.Job, payload interface{}) error
}
