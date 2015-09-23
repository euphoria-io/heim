package psql

import (
	"database/sql"
	"fmt"
	"time"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/jobs"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"

	"gopkg.in/gorp.v1"
)

type Email struct {
	ID        string
	AccountID string `db:"account_id"`
	JobID     int64  `db:"job_id"`
	EmailType string `db:"email_type"`
	SendTo    string `db:"send_to"`
	SendFrom  string `db:"send_from"`
	Message   []byte
	Created   time.Time
	Delivered gorp.NullTime
	Failed    gorp.NullTime
}

func (e *Email) ToBackend() (*emails.EmailRef, error) {
	ref := &emails.EmailRef{
		ID:        e.ID,
		JobID:     snowflake.Snowflake(e.JobID),
		EmailType: e.EmailType,
		SendTo:    e.SendTo,
		SendFrom:  e.SendFrom,
		Message:   e.Message,
		Created:   e.Created,
	}

	if err := ref.AccountID.FromString(e.AccountID); err != nil {
		return nil, err
	}

	if e.Delivered.Valid {
		ref.Delivered = e.Delivered.Time
	}
	if e.Failed.Valid {
		ref.Failed = e.Failed.Time
	}
	return ref, nil
}

func (e *Email) FromBackend(ref *emails.EmailRef) {
	e.ID = ref.ID
	e.AccountID = ref.AccountID.String()
	e.JobID = int64(ref.JobID)
	e.EmailType = ref.EmailType
	e.SendTo = ref.SendTo
	e.SendFrom = ref.SendFrom
	e.Message = ref.Message
	e.Created = ref.Created

	if ref.Delivered.IsZero() {
		e.Delivered.Valid = false
	} else {
		e.Delivered.Valid = true
		e.Delivered.Time = ref.Delivered
	}

	if ref.Failed.IsZero() {
		e.Failed.Valid = false
	} else {
		e.Failed.Valid = true
		e.Failed.Time = ref.Failed
	}
}

type EmailTracker struct {
	Backend *Backend
}

func (et *EmailTracker) Send(
	ctx scope.Context, js jobs.JobService, templater *templates.Templater, deliverer emails.Deliverer,
	account proto.Account, templateName string, data interface{}) (
	*emails.EmailRef, error) {

	// choose a Message-ID
	sf, err := snowflake.New()
	if err != nil {
		return nil, err
	}
	msgID := fmt.Sprintf("<%s@%s>", sf, deliverer.LocalName())

	// choose an address to send to
	to := ""
	/*
	   requireVerifiedAddress := true
	   switch templateName {
	   case proto.WelcomeEmail, proto.RoomInvitationWelcomeEmail, proto.PasswordResetEmail:
	       requireVerifiedAddress = false
	   }
	*/
	for _, pid := range account.PersonalIdentities() {
		if pid.Namespace() == "email" {
			/*
			   if !pid.Verified() && requireVerifiedAddress {
			       continue
			   }
			*/
			to = pid.ID()
			break
		}
	}
	if to == "" {
		fmt.Printf("no email address to deliver to\n")
		return nil, fmt.Errorf("account has no email address to deliver %s to", templateName)
	}

	// construct the email
	ref, err := emails.NewEmail(templater, msgID, to, templateName, data)
	if err != nil {
		return nil, err
	}
	ref.AccountID = account.ID()

	// get underlying JobQueue so we can add-and-claim in the same transaction as the email insert
	abstractQueue, err := js.GetQueue(ctx, jobs.EmailQueue)
	if err != nil {
		return nil, err
	}
	jq := abstractQueue.(*JobQueueBinding)

	t, err := et.Backend.DbMap.Begin()
	if err != nil {
		return nil, err
	}

	// insert job first, so we know what JobID to associate with the email when we insert it
	job, err := jq.addAndClaim(
		ctx, t, jobs.EmailJobType, &jobs.EmailJob{EmailID: ref.ID}, "immediate", jobs.EmailJobOptions...)
	if err != nil {
		rollback(ctx, t)
		return nil, err
	}
	ref.JobID = job.ID

	// insert the email
	var email Email
	email.FromBackend(ref)
	if err := t.Insert(&email); err != nil {
		rollback(ctx, t)
		return nil, err
	}

	// finalize and spin off first delivery attempt
	if err := t.Commit(); err != nil {
		return nil, err
	}

	child := ctx.Fork()
	child.WaitGroup().Add(1)
	go func() {
		defer child.WaitGroup().Done()

		fmt.Printf("delivering to %s\n", to)
		if err := deliverer.Deliver(ctx, ref); err != nil {
			if jerr := job.Fail(ctx, err.Error()); jerr != nil {
				backend.Logger(ctx).Printf("error reporting job %s failure: %s", job.ID, jerr)
			}
			return
		}
		if err := job.Complete(ctx); err != nil {
			backend.Logger(ctx).Printf("error reporting job %s completion: %s", job.ID, err)
		}

		if _, err := et.Backend.DbMap.Exec("UPDATE email SET delivered = $2 WHERE id = $1", ref.ID, ref.Delivered); err != nil {
			backend.Logger(ctx).Printf("error marking email %s as delivered: %s", ref.ID, err)
		}
	}()

	return ref, nil
}

func (et *EmailTracker) Get(accountID snowflake.Snowflake, id string) (*emails.EmailRef, error) {
	row, err := et.Backend.DbMap.Get(Email{}, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrEmailNotFound
		}
		return nil, err
	}

	ref, err := row.(*Email).ToBackend()
	if err != nil {
		return nil, err
	}

	if ref.AccountID != accountID {
		return nil, proto.ErrEmailNotFound
	}

	return ref, nil
}

func (et *EmailTracker) List(accountID snowflake.Snowflake, n int, before time.Time) ([]*emails.EmailRef, error) {
	return nil, notImpl
}
