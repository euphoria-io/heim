package worker

import (
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/jobs"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type EmailWorker struct {
	d  emails.Deliverer
	et proto.EmailTracker
}

func (EmailWorker) QueueName() string     { return jobs.EmailQueue }
func (EmailWorker) JobType() jobs.JobType { return jobs.EmailJobType }

func (w *EmailWorker) Init(heim *proto.Heim) error {
	w.d = heim.EmailDeliverer
	w.et = heim.Backend.EmailTracker()
	return nil
}

func (w *EmailWorker) Work(ctx scope.Context, job *jobs.Job, payload interface{}) error {
	emailJob := payload.(*jobs.EmailJob)
	return w.send(ctx, emailJob.AccountID, emailJob.EmailID)
}

func (w *EmailWorker) send(ctx scope.Context, accountID snowflake.Snowflake, msgID string) error {
	ref, err := w.et.Get(ctx, accountID, msgID)
	if err != nil {
		return err
	}

	if err := w.d.Deliver(ctx, ref); err != nil {
		return err
	}

	if err := w.et.MarkDelivered(ctx, accountID, msgID); err != nil {
		// We failed to mark the email as delivered, which is unfortunate,
		// but not quite as unfortunate as delivering it twice would be.
		// So we swallow the error here but log it noisily.
		logging.Logger(ctx).Printf("failed to mark email %s/%s as delivered: %s", accountID, msgID, err)
	}

	return nil
}

func init() {
	register(&EmailWorker{})
}
