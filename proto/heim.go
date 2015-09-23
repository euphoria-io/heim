package proto

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"

	"encoding/hex"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto/emails"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/templates"
	"euphoria.io/scope"
)

type Heim struct {
	Backend    Backend
	Cluster    cluster.Cluster
	PeerDesc   *cluster.PeerDesc
	Context    scope.Context
	KMS        security.KMS
	StaticPath string

	EmailTemplater *templates.Templater
	EmailDeliverer emails.Deliverer
}

func (heim *Heim) MockDeliverer() emails.MockDeliverer {
	return heim.EmailDeliverer.(emails.MockDeliverer)
}

func (heim *Heim) SendEmail(
	ctx scope.Context, b Backend, account Account, templateName string, data interface{}) (*emails.EmailRef, error) {

	return b.EmailTracker().Send(ctx, b.Jobs(), heim.EmailTemplater, heim.EmailDeliverer, account, templateName, data)
}

func (heim *Heim) OnAccountPasswordChanged(ctx scope.Context, b Backend, account Account) error {
	// TODO: account names
	params := &PasswordChangedEmailParams{
		CommonEmailParams: DefaultCommonEmailParams,
		AccountName:       account.Name(),
	}
	if _, err := heim.SendEmail(ctx, b, account, PasswordChangedEmail, params); err != nil {
		return err
	}

	return nil
}

func (heim *Heim) OnAccountPasswordResetRequest(
	ctx scope.Context, b Backend, account Account, req *PasswordResetRequest) error {

	// TODO: account names
	params := &PasswordResetEmailParams{
		CommonEmailParams: DefaultCommonEmailParams,
		AccountName:       account.Name(),
		Confirmation:      req.String(),
	}
	if _, err := heim.SendEmail(ctx, b, account, PasswordResetEmail, params); err != nil {
		return err
	}

	return nil
}

func (heim *Heim) OnAccountRegistration(
	ctx scope.Context, b Backend, account Account, clientKey *security.ManagedKey) error {

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
		userKey := account.UserKey()
		if err := userKey.Decrypt(clientKey); err != nil {
			return err
		}

		token, err := emailVerificationToken(&userKey, email)
		if err != nil {
			return fmt.Errorf("verification token: %s", err)
		}

		params := &WelcomeEmailParams{
			CommonEmailParams: DefaultCommonEmailParams,
			VerificationToken: hex.EncodeToString(token),
		}
		if _, err := heim.SendEmail(ctx, b, account, WelcomeEmail, params); err != nil {
			return err
		}
	}

	return nil
}

func emailVerificationToken(key *security.ManagedKey, email string) ([]byte, error) {
	if key.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	mac := hmac.New(sha1.New, key.Plaintext)
	mac.Write([]byte(email))
	return mac.Sum(nil), nil
}

func CheckEmailVerificationToken(kms security.KMS, account Account, email string, token []byte) error {
	systemKey := account.SystemKey()
	if err := kms.DecryptKey(&systemKey); err != nil {
		return err
	}

	expected, err := emailVerificationToken(&systemKey, email)
	if err != nil {
		return err
	}

	if !hmac.Equal(token, expected) {
		return ErrInvalidVerificationToken
	}

	return nil
}
