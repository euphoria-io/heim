package console

import (
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

func init() {
	register("grant-staff", grantStaff{})
	register("revoke-staff", revokeStaff{})
}

type grantStaff struct{}

func (grantStaff) usage() string { return "usage: grant-staff ACCOUNT KMSTYPE [CREDENTIALS]" }

func (grantStaff) run(ctx scope.Context, c *console, args []string) error {
	if len(args) < 2 {
		return usageError("account and kms type must be given")
	}

	kmsType := security.KMSType(args[1])
	kmsCred, err := kmsType.KMSCredential()
	if err != nil {
		return err
	}

	if len(args) < 3 {
		if kmsType != security.LocalKMSType {
			return usageError("kms type %s requires credentials to be provided", kmsType)
		}
		mockKMS, ok := c.kms.(security.MockKMS)
		if !ok {
			return usageError("this backend does not support kms type %s", kmsType)
		}
		kmsCred = mockKMS.KMSCredential()
	} else {
		if err := kmsCred.UnmarshalJSON([]byte(args[2])); err != nil {
			return err
		}
	}

	account, err := c.resolveAccount(ctx, args[0])
	if err != nil {
		return err
	}

	c.Printf("Granting staff capability to account %s\n", account.ID())
	return c.backend.AccountManager().GrantStaff(ctx, account.ID(), kmsCred)
}

type revokeStaff struct{}

func (revokeStaff) usage() string { return "usage: revoke-staff ACCOUNT" }

func (revokeStaff) run(ctx scope.Context, c *console, args []string) error {
	if len(args) < 1 {
		return usageError("account must be given")
	}

	account, err := c.resolveAccount(ctx, args[0])
	if err != nil {
		return err
	}

	if !account.IsStaff() {
		c.Printf("NOTE: this account isn't currently holding a staff capability\n")
	}
	c.Printf("revoking staff capability from %s\n", account.ID())
	return c.backend.AccountManager().RevokeStaff(ctx, account.ID())
}
