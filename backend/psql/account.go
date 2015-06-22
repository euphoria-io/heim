package psql

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
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
	EncryptedSystemKey  []byte         `db:"encrypted_system_key"`
	EncryptedUserKey    []byte         `db:"encrypted_user_key"`
	EncryptedPrivateKey []byte         `db:"encrypted_private_key"`
	PublicKey           []byte         `db:"public_key"`
	StaffCapabilityID   sql.NullString `db:"staff_capability_id"`
}

func (a *Account) Bind(b *Backend) *AccountBinding {
	return &AccountBinding{
		Backend: b,
		Account: a,
	}
}

type AccountWithStaffCapability struct {
	AccountID            string         `db:"id"`
	AccountNonce         []byte         `db:"nonce"`
	StaffCapabilityID    sql.NullString `db:"staff_capability_id"`
	StaffCapabilityNonce []byte         `db:"staff_capability_nonce"`
	Account
	Capability
}

func (awsc *AccountWithStaffCapability) Bind(b *Backend) *AccountBinding {
	awsc.Account.ID = awsc.AccountID
	awsc.Account.Nonce = awsc.AccountNonce
	ab := awsc.Account.Bind(b)
	if awsc.StaffCapabilityID.Valid {
		awsc.Capability.ID = awsc.StaffCapabilityID.String
		awsc.Capability.NonceBytes = awsc.StaffCapabilityNonce
		ab.StaffCapability = &awsc.Capability
	}
	return ab
}

type PersonalIdentity struct {
	Namespace string
	ID        string
	AccountID string `db:"account_id"`
}

type AccountBinding struct {
	*Backend
	*Account
	StaffCapability *Capability
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

func (ab *AccountBinding) IsStaff() bool { return ab.StaffCapability != nil }

func (ab *AccountBinding) UnlockStaffKMS(clientKey *security.ManagedKey) (security.KMS, error) {
	if ab.StaffCapability == nil {
		return nil, proto.ErrAccessDenied
	}

	iv := make([]byte, proto.ClientKeyType.BlockSize())
	copy(iv, ab.Account.Nonce)
	key := &security.ManagedKey{
		KeyType:    proto.ClientKeyType,
		IV:         iv,
		Ciphertext: ab.Account.EncryptedUserKey,
	}
	if err := key.Decrypt(clientKey); err != nil {
		return nil, err
	}

	ssc := &security.SharedSecretCapability{Capability: ab.StaffCapability}
	data, err := ssc.DecryptPayload(key)
	if err != nil {
		return nil, err
	}

	var kmsType security.KMSType
	if err := json.Unmarshal(ssc.PublicPayload(), &kmsType); err != nil {
		return nil, err
	}

	kmsCred, err := kmsType.KMSCredential()
	if err != nil {
		return nil, err
	}

	if err := kmsCred.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return kmsCred.KMS(), nil
}

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

	var row AccountWithStaffCapability
	err := b.DbMap.SelectOne(
		&row,
		"SELECT a.id, a.nonce, a.mac, a.encrypted_system_key, a.encrypted_user_key,"+
			" a.encrypted_private_key, a.public_key,"+
			" c.id AS staff_capability_id, c.nonce AS staff_capability_nonce,"+
			" c.encrypted_private_data, c.public_data"+
			" FROM (account a JOIN personal_identity i ON a.id = i.account_id)"+
			" LEFT OUTER JOIN capability c ON a.staff_capability_id = c.id"+
			" WHERE i.namespace = $1 AND i.id = $2 AND i.account_id = a.id",
		namespace, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrAccountNotFound
		}
		return nil, err
	}

	return row.Bind(b.Backend), nil
}

func (b *AccountManagerBinding) Get(
	ctx scope.Context, id snowflake.Snowflake) (proto.Account, error) {

	var row AccountWithStaffCapability
	err := b.DbMap.SelectOne(
		&row,
		"SELECT a.id, a.nonce, a.mac, a.encrypted_system_key, a.encrypted_user_key,"+
			" a.encrypted_private_key, a.public_key,"+
			" c.id AS staff_capability_id, c.nonce AS staff_capability_nonce,"+
			" c.encrypted_private_data, c.public_data"+
			" FROM account a LEFT OUTER JOIN capability c ON a.staff_capability_id = c.id"+
			" WHERE a.id = $1",
		id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrAccountNotFound
		}
		return nil, err
	}

	return row.Bind(b.Backend), nil
}

func (b *AccountManagerBinding) GrantStaff(
	ctx scope.Context, accountID snowflake.Snowflake, kmsCred security.KMSCredential) error {

	// Look up the target account's (system) encrypted client key. This is
	// not part of the transaction, because we want to interact with KMS
	// before we proceed. That should be fine, since this is an infrequently
	// used action.
	var row struct {
		EncryptedClientKey []byte `db:"encrypted_system_key"`
		Nonce              []byte `db:"nonce"`
	}
	err := b.DbMap.SelectOne(
		&row, "SELECT encrypted_system_key, nonce FROM account WHERE id = $1", accountID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return proto.ErrAccountNotFound
		}
		return err
	}

	// Use kmsCred to obtain kms and decrypt the client's key.
	kms := kmsCred.KMS()
	clientKey := &security.ManagedKey{
		KeyType:      proto.ClientKeyType,
		Ciphertext:   row.EncryptedClientKey,
		ContextKey:   "nonce",
		ContextValue: base64.URLEncoding.EncodeToString(row.Nonce),
	}
	if err := kms.DecryptKey(clientKey); err != nil {
		return err
	}

	// Grant staff capability. This involves marshalling kmsCred to JSON and
	// encrypting it with the client key.
	nonce, err := kms.GenerateNonce(clientKey.KeyType.BlockSize())
	if err != nil {
		return err
	}

	capability, err := security.GrantSharedSecretCapability(clientKey, nonce, kmsCred.KMSType(), kmsCred)
	if err != nil {
		return err
	}

	// Store capability and update account table.
	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	dbCap := &Capability{
		ID:                   capability.CapabilityID(),
		NonceBytes:           capability.Nonce(),
		EncryptedPrivateData: capability.EncryptedPayload(),
		PublicData:           capability.PublicPayload(),
	}
	if err := t.Insert(dbCap); err != nil {
		rollback()
		return err
	}

	result, err := t.Exec(
		"UPDATE account SET staff_capability_id = $2 WHERE id = $1",
		accountID.String(), capability.CapabilityID())
	if err != nil {
		rollback()
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		rollback()
		return err
	}
	if n != 1 {
		rollback()
		return proto.ErrAccountNotFound
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}

func (b *AccountManagerBinding) RevokeStaff(ctx scope.Context, accountID snowflake.Snowflake) error {
	_, err := b.DbMap.Exec(
		"DELETE FROM capability USING account"+
			" WHERE account.id = $1 AND capability.id = account.staff_capability_id",
		accountID.String())
	return err
}
