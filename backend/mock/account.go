package mock

import (
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
)

func NewAccount(kms security.KMS, password string) (proto.Account, *security.ManagedKey, error) {
	id, err := snowflake.New()
	if err != nil {
		return nil, nil, err
	}

	sec, clientKey, err := proto.NewAccountSecurity(kms, password)
	if err != nil {
		return nil, nil, err
	}

	account := &memAccount{
		id:  id,
		sec: *sec,
	}
	return account, clientKey, nil
}

type memAccount struct {
	id    snowflake.Snowflake
	sec   proto.AccountSecurity
	staff bool
}

func (a *memAccount) ID() snowflake.Snowflake { return a.id }

func (a *memAccount) KeyFromPassword(password string) *security.ManagedKey {
	return security.KeyFromPasscode([]byte(password), a.sec.Nonce, a.sec.UserKey.KeyType)
}

func (a *memAccount) KeyPair() security.ManagedKeyPair { return a.sec.KeyPair.Clone() }

func (a *memAccount) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	return a.sec.Unlock(clientKey)
}

func (a *memAccount) IsStaff() bool { return a.staff }
