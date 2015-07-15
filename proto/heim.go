package proto

import (
	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type Heim struct {
	Backend Backend
	Cluster cluster.Cluster
	Context scope.Context
	Emailer emails.Emailer
	KMS     security.KMS
}

func (heim *Heim) OnAccountRegistration(ctx scope.Context, account Account) error {
	// Pick an email identity.
	email := ""
	verified := false
	for _, ident := range account.PersonalIdentities() {
		if ident.Namespace() == "email" {
			if email == "" {
				email = ident.ID()
			}
			if ident.Verified() {
				verified = true
			}
		}
	}

	// If an email is found but no email is verified, send a welcome email.
	if email != "" && !verified {
		_, err := heim.Emailer.Send(ctx, email, WelcomeEmail, map[string]interface{}{"account": account})
		if err != nil {
			return err
		}
	}

	return nil
}
