package psql

import (
	"database/sql"
	"encoding/base64"
	"strings"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type Account struct {
	ID                  string
	Nonce               []byte
	MAC                 []byte
	EncryptedSystemKey  []byte `db:"encrypted_system_key"`
	EncryptedUserKey    []byte `db:"encrypted_user_key"`
	EncryptedPrivateKey []byte `db:"encrypted_private_key"`
	PublicKey           []byte `db:"public_key"`
	Staff               bool
}

func (a *Account) Bind(b *Backend) *AccountBinding {
	return &AccountBinding{
		Backend: b,
		Account: a,
	}
}

type PersonalIdentity struct {
	Namespace string
	ID        string
	AccountID string `db:"account_id"`
}

type AccountBinding struct {
	*Backend
	*Account
}

func (ab *AccountBinding) ID() snowflake.Snowflake {
	var id snowflake.Snowflake
	_ = id.FromString(ab.Account.ID)
	return id
}

func (ab *AccountBinding) KeyFromPassword(password string) *security.ManagedKey {
	return security.KeyFromPasscode([]byte(password), ab.Account.Nonce, proto.ClientKeyType)
}

func (ab *AccountBinding) KeyPair() security.ManagedKeyPair {
	iv := make([]byte, proto.ClientKeyType.BlockSize())
	copy(iv, ab.Account.Nonce)

	return security.ManagedKeyPair{
		KeyPairType:         security.Curve25519,
		IV:                  iv,
		EncryptedPrivateKey: ab.Account.EncryptedPrivateKey,
		PublicKey:           ab.Account.PublicKey,
	}
}

func (ab *AccountBinding) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	iv := make([]byte, proto.ClientKeyType.BlockSize())
	copy(iv, ab.Account.Nonce)

	sec := &proto.AccountSecurity{
		Nonce: ab.Account.Nonce,
		MAC:   ab.Account.MAC,
		SystemKey: security.ManagedKey{
			KeyType:      proto.ClientKeyType,
			Ciphertext:   ab.Account.EncryptedSystemKey,
			ContextKey:   "nonce",
			ContextValue: base64.URLEncoding.EncodeToString(ab.Account.Nonce),
		},
		UserKey: security.ManagedKey{
			KeyType:    proto.ClientKeyType,
			IV:         iv,
			Ciphertext: ab.Account.EncryptedUserKey,
		},
		KeyPair: security.ManagedKeyPair{
			KeyPairType:         security.Curve25519,
			IV:                  iv,
			EncryptedPrivateKey: ab.Account.EncryptedPrivateKey,
			PublicKey:           ab.Account.PublicKey,
		},
	}
	return sec.Unlock(clientKey)
}

func (ab *AccountBinding) IsStaff() bool { return ab.Staff }

type AccountManagerBinding struct {
	*Backend
}

func (b *AccountManagerBinding) Register(
	ctx scope.Context, kms security.KMS, namespace, id, password string,
	agentID string, agentKey *security.ManagedKey) (
	proto.Account, *security.ManagedKey, error) {

	// Generate ID for new account.
	accountID, err := snowflake.New()
	if err != nil {
		return nil, nil, err
	}

	// Generate credentials in advance of working in DB transaction.
	backend.Logger(ctx).Printf("NewAccountSecurity: kms=%#v", kms)
	sec, clientKey, err := proto.NewAccountSecurity(kms, password)
	if err != nil {
		return nil, nil, err
	}

	// Begin transaction to check on identity availability and store new account data.
	t, err := b.DbMap.Begin()
	if err != nil {
		return nil, nil, err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	// Insert new rows for account.
	account := &Account{
		ID:                  accountID.String(),
		Nonce:               sec.Nonce,
		MAC:                 sec.MAC,
		EncryptedSystemKey:  sec.SystemKey.Ciphertext,
		EncryptedUserKey:    sec.UserKey.Ciphertext,
		EncryptedPrivateKey: sec.KeyPair.EncryptedPrivateKey,
		PublicKey:           sec.KeyPair.PublicKey,
	}
	personalIdentity := &PersonalIdentity{
		Namespace: namespace,
		ID:        id,
		AccountID: accountID.String(),
	}
	if err := t.Insert(account, personalIdentity); err != nil {
		rollback()
		if strings.HasPrefix(err.Error(), "pq: duplicate key value") {
			return nil, nil, proto.ErrPersonalIdentityInUse
		}
		return nil, nil, err
	}

	// Look up the associated agent.
	atb := &AgentTrackerBinding{b.Backend}
	agent, err := atb.getFromDB(agentID, t)
	if err != nil {
		rollback()
		return nil, nil, err
	}
	if err := agent.SetClientKey(agentKey, clientKey); err != nil {
		rollback()
		return nil, nil, err
	}
	err = atb.setClientKeyInDB(agentID, accountID.String(), agent.EncryptedClientKey.Ciphertext, t)
	if err != nil {
		rollback()
		return nil, nil, err
	}

	// Commit the transaction.
	if err := t.Commit(); err != nil {
		return nil, nil, err
	}
	backend.Logger(ctx).Printf("registered new account %s for %s:%s", account.ID, namespace, id)

	return account.Bind(b.Backend), clientKey, nil
}

func (b *AccountManagerBinding) Resolve(
	ctx scope.Context, namespace, id string) (proto.Account, error) {

	var acc Account
	err := b.DbMap.SelectOne(
		&acc,
		"SELECT a.id, a.nonce, a.mac, a.encrypted_system_key, a.encrypted_user_key,"+
			" a.encrypted_private_key, a.public_key"+
			" FROM account a, personal_identity i"+
			" WHERE i.namespace = $1 AND i.id = $2 AND i.account_id = a.id",
		namespace, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrAccountNotFound
		}
		return nil, err
	}
	return acc.Bind(b.Backend), nil
}

func (b *AccountManagerBinding) Get(
	ctx scope.Context, id snowflake.Snowflake) (proto.Account, error) {

	row, err := b.DbMap.Get(Account{}, id.String())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, proto.ErrAccountNotFound
	}
	return row.(*Account).Bind(b.Backend), nil
}

func (b *AccountManagerBinding) SetStaff(
	ctx scope.Context, accountID snowflake.Snowflake, isStaff bool) error {

	result, err := b.DbMap.Exec(
		"UPDATE account SET staff = $2 WHERE id = $1", accountID.String(), isStaff)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return proto.ErrAccountNotFound
	}
	return nil
}
