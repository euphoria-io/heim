package mock

import (
	"fmt"
	"sync"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/jobs"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

type EmailTracker struct {
	m               sync.Mutex
	emailsByAccount map[snowflake.Snowflake][]*emails.EmailRef
}

func (et *EmailTracker) Send(
	ctx scope.Context, js jobs.JobService, templater *templates.Templater, deliverer emails.Deliverer,
	account proto.Account, templateName string, data interface{}) (
	*emails.EmailRef, error) {

	sf, err := snowflake.New()
	if err != nil {
		return nil, err
	}
	msgID := fmt.Sprintf("<%s@%s>", sf, deliverer.LocalName())

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

	ref, err := emails.NewEmail(templater, msgID, to, templateName, data)
	if err != nil {
		return nil, err
	}
	ref.AccountID = account.ID()

	jq, err := js.GetQueue(ctx, jobs.EmailQueue)
	if err != nil {
		return nil, err
	}

	payload := &jobs.EmailJob{
		AccountID: account.ID(),
		EmailID:   ref.ID,
	}
	job, err := jq.AddAndClaim(ctx, jobs.EmailJobType, payload, "immediate", jobs.EmailJobOptions...)
	if err != nil {
		return nil, err
	}

	ref.JobID = job.ID

	et.m.Lock()
	if et.emailsByAccount == nil {
		et.emailsByAccount = map[snowflake.Snowflake][]*emails.EmailRef{}
	}
	et.emailsByAccount[account.ID()] = append(et.emailsByAccount[account.ID()], ref)
	et.m.Unlock()

	child := ctx.Fork()
	child.WaitGroup().Add(1)
	go func() {
		defer child.WaitGroup().Done()

		fmt.Printf("delivering to %s\n", to)
		if err := deliverer.Deliver(ctx, ref); err != nil {
			if jerr := job.Fail(ctx, err.Error()); jerr != nil {
				logging.Logger(ctx).Printf("error reporting job failure: %s", jerr)
			}
			return
		}
		if err := job.Complete(ctx); err != nil {
			logging.Logger(ctx).Printf("error reporting job completion: %s", err)
		}
	}()

	return ref, nil
}

func (et *EmailTracker) get(accountID snowflake.Snowflake, id string) (*emails.EmailRef, error) {
	for _, ref := range et.emailsByAccount[accountID] {
		if ref.ID == id {
			return ref, nil
		}
	}
	return nil, proto.ErrEmailNotFound
}

func (et *EmailTracker) Get(ctx scope.Context, accountID snowflake.Snowflake, id string) (*emails.EmailRef, error) {
	et.m.Lock()
	defer et.m.Unlock()
	return et.get(accountID, id)
}

func (et *EmailTracker) List(ctx scope.Context, accountID snowflake.Snowflake, n int, before time.Time) ([]*emails.EmailRef, error) {
	et.m.Lock()
	defer et.m.Unlock()

	refs := et.emailsByAccount[accountID]
	var i int
	for i = len(refs) - 1; i >= 0; i-- {
		if refs[i].Created.Before(before) {
			break
		}
	}
	i += 1
	j := i - n
	if j < 0 {
		j = 0
	}
	return refs[i:j], nil
}

func (et *EmailTracker) MarkDelivered(ctx scope.Context, accountID snowflake.Snowflake, id string) error {
	et.m.Lock()
	defer et.m.Unlock()

	ref, err := et.get(accountID, id)
	if err != nil {
		return err
	}

	if !ref.Delivered.IsZero() {
		return proto.ErrEmailAlreadyDelivered
	}

	ref.Delivered = time.Now()
	return nil
}
