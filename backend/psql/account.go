package psql

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"gopkg.in/gorp.v1"
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
	CapabilityAccountID  sql.NullString `db:"account_id"`
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
	Verified  bool
}

type PersonalIdentityBinding struct {
	pid *PersonalIdentity
}

func (pib *PersonalIdentityBinding) Namespace() string { return pib.pid.Namespace }
func (pib *PersonalIdentityBinding) ID() string        { return pib.pid.ID }
func (pib *PersonalIdentityBinding) Verified() bool    { return pib.pid.Verified }

type PasswordResetRequest struct {
	ID          string
	AccountID   string `db:"account_id"`
	Key         []byte
	Requested   time.Time
	Expires     time.Time
	Consumed    gorp.NullTime
	Invalidated gorp.NullTime
}

type AccountBinding struct {
	*Backend
	*Account
	StaffCapability *Capability
	identities      []proto.PersonalIdentity
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

func (ab *AccountBinding) UserKey() security.ManagedKey {
	iv := make([]byte, proto.ClientKeyType.BlockSize())
	copy(iv, ab.Account.Nonce)

	key := &security.ManagedKey{
		KeyType:    proto.ClientKeyType,
		IV:         iv,
		Ciphertext: ab.Account.EncryptedUserKey,
	}
	return key.Clone()
}

func (ab *AccountBinding) SystemKey() security.ManagedKey {
	key := &security.ManagedKey{
		KeyType:      proto.ClientKeyType,
		Ciphertext:   ab.Account.EncryptedSystemKey,
		ContextKey:   "nonce",
		ContextValue: base64.URLEncoding.EncodeToString(ab.Account.Nonce),
	}
	return key.Clone()
}

func (ab *AccountBinding) accountSecurity() *proto.AccountSecurity {
	iv := make([]byte, proto.ClientKeyType.BlockSize())
	copy(iv, ab.Account.Nonce)

	return &proto.AccountSecurity{
		Nonce: ab.Account.Nonce,
		MAC:   ab.Account.MAC,
		UserKey: security.ManagedKey{
			KeyType:    proto.ClientKeyType,
			IV:         iv,
			Ciphertext: ab.Account.EncryptedUserKey,
		},
		SystemKey: security.ManagedKey{
			KeyType:      proto.ClientKeyType,
			Ciphertext:   ab.Account.EncryptedSystemKey,
			ContextKey:   "nonce",
			ContextValue: base64.URLEncoding.EncodeToString(ab.Account.Nonce),
		},
		KeyPair: security.ManagedKeyPair{
			KeyPairType:         security.Curve25519,
			IV:                  iv,
			EncryptedPrivateKey: ab.Account.EncryptedPrivateKey,
			PublicKey:           ab.Account.PublicKey,
		},
	}
}

func (ab *AccountBinding) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	return ab.accountSecurity().Unlock(clientKey)
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
		Ciphertext: make([]byte, len(ab.Account.EncryptedUserKey)),
	}
	copy(key.Ciphertext, ab.Account.EncryptedUserKey)
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

func (ab *AccountBinding) PersonalIdentities() []proto.PersonalIdentity { return ab.identities }

type AccountManagerBinding struct {
	*Backend
}

func (b *AccountManagerBinding) VerifyPersonalIdentity(ctx scope.Context, namespace, id string) error {
	res, err := b.DbMap.Exec(
		"UPDATE personal_identity SET verified = true WHERE namespace = $1 and id = $2",
		namespace, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return proto.ErrAccountNotFound
		}
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return proto.ErrAccountNotFound
	}

	return nil
}

func (b *AccountManagerBinding) ChangeClientKey(
	ctx scope.Context, accountID snowflake.Snowflake, oldKey, newKey *security.ManagedKey) error {

	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	row, err := t.Get(Account{}, accountID.String())
	if err != nil {
		rollback()
		if err == sql.ErrNoRows {
			return proto.ErrAccountNotFound
		}
		return err
	}
	account := row.(*Account)

	sec := account.Bind(b.Backend).accountSecurity()
	if err := sec.ChangeClientKey(oldKey, newKey); err != nil {
		rollback()
		return err
	}

	res, err := t.Exec(
		"UPDATE account SET mac = $2, encrypted_user_key = $3 WHERE id = $1",
		accountID.String(), sec.MAC, sec.UserKey.Ciphertext)
	if err != nil {
		rollback()
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		rollback()
		return err
	}
	if n == 0 {
		rollback()
		return proto.ErrAccountNotFound
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}

func (b *AccountManagerBinding) SetUserKey(
	ctx scope.Context, accountID snowflake.Snowflake, key *security.ManagedKey) error {

	if !key.Encrypted() {
		return security.ErrKeyMustBeEncrypted
	}

	res, err := b.DbMap.Exec(
		"UPDATE account SET encrypted_user_key = $2 WHERE id = $1", accountID.String(), key.Ciphertext)
	if err != nil {
		if err == sql.ErrNoRows {
			return proto.ErrAccountNotFound
		}
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return proto.ErrAccountNotFound
	}

	return nil
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
	if err := t.Insert(account); err != nil {
		rollback()
		return nil, nil, err
	}
	if err := t.Insert(personalIdentity); err != nil {
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

	ab := account.Bind(b.Backend)
	ab.identities = []proto.PersonalIdentity{&PersonalIdentityBinding{personalIdentity}}
	return ab, clientKey, nil
}

func (b *AccountManagerBinding) Resolve(ctx scope.Context, namespace, id string) (proto.Account, error) {
	t, err := b.DbMap.Begin()
	account, err := b.resolve(t, namespace, id)
	if err != nil {
		if rerr := t.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
		return nil, err
	}
	if err := t.Commit(); err != nil {
		return nil, err
	}
	return account, nil
}

func (b *AccountManagerBinding) resolve(
	db gorp.SqlExecutor, namespace, id string) (*AccountBinding, error) {

	var pid PersonalIdentity
	err := db.SelectOne(
		&pid,
		"SELECT account_id FROM personal_identity WHERE namespace = $1 AND id = $2",
		namespace, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrAccountNotFound
		}
		return nil, err
	}

	var accountID snowflake.Snowflake
	if err := accountID.FromString(pid.AccountID); err != nil {
		return nil, err
	}

	return b.get(db, accountID)
}

func (b *AccountManagerBinding) Get(ctx scope.Context, id snowflake.Snowflake) (proto.Account, error) {
	t, err := b.DbMap.Begin()
	account, err := b.get(t, id)
	if err != nil {
		if rerr := t.Rollback(); rerr != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
		return nil, err
	}
	if err := t.Commit(); err != nil {
		return nil, err
	}
	return account, nil
}

func (b *AccountManagerBinding) get(
	db gorp.SqlExecutor, id snowflake.Snowflake) (*AccountBinding, error) {

	accountCols, err := allColumns(b.DbMap, Account{}, "a")
	if err != nil {
		return nil, err
	}

	capabilityCols, err := allColumns(b.DbMap, Capability{}, "c",
		"ID", "staff_capability_id",
		"nonce", "staff_capability_nonce")
	if err != nil {
		return nil, err
	}

	var row AccountWithStaffCapability
	err = db.SelectOne(
		&row,
		fmt.Sprintf("SELECT %s, %s FROM account a LEFT OUTER JOIN capability c ON a.staff_capability_id = c.id WHERE a.id = $1",
			accountCols, capabilityCols),
		id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, proto.ErrAccountNotFound
		}
		return nil, err
	}

	ab := row.Bind(b.Backend)

	piCols, err := allColumns(b.DbMap, PersonalIdentity{}, "")
	if err != nil {
		return nil, err
	}
	rows, err := db.Select(PersonalIdentity{}, fmt.Sprintf("SELECT %s FROM personal_identity WHERE account_id = $1", piCols), id.String())
	switch err {
	case sql.ErrNoRows:
	case nil:
		ab.identities = make([]proto.PersonalIdentity, len(rows))
		for i, row := range rows {
			ab.identities[i] = &PersonalIdentityBinding{row.(*PersonalIdentity)}
		}
	default:
		return nil, err
	}

	return ab, nil
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

func (b *AccountManagerBinding) RequestPasswordReset(
	ctx scope.Context, kms security.KMS, namespace, id string) (
	proto.Account, *proto.PasswordResetRequest, error) {

	t, err := b.DbMap.Begin()
	if err != nil {
		return nil, nil, err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	account, err := b.resolve(t, namespace, id)
	if err != nil {
		rollback()
		return nil, nil, err
	}

	req, err := proto.GeneratePasswordResetRequest(kms, account.ID())
	if err != nil {
		rollback()
		return nil, nil, err
	}

	stored := &PasswordResetRequest{
		ID:        req.ID.String(),
		AccountID: req.AccountID.String(),
		Key:       req.Key,
		Requested: req.Requested,
		Expires:   req.Expires,
	}
	if err := t.Insert(stored); err != nil {
		rollback()
		return nil, nil, err
	}

	if err := t.Commit(); err != nil {
		rollback()
		return nil, nil, err
	}

	return account, req, nil
}

func (b *AccountManagerBinding) ConfirmPasswordReset(
	ctx scope.Context, kms security.KMS, confirmation, password string) error {

	id, mac, err := proto.ParsePasswordResetConfirmation(confirmation)
	if err != nil {
		return err
	}

	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			backend.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	req := &proto.PasswordResetRequest{
		ID: id,
	}

	var (
		stored  PasswordResetRequest
		account *AccountBinding
	)

	cols, err := allColumns(b.DbMap, PasswordResetRequest{}, "")
	if err != nil {
		return err
	}
	err = t.SelectOne(
		&stored,
		fmt.Sprintf(
			"SELECT %s FROM password_reset_request WHERE id = $1 AND expires > NOW() AND invalidated IS NULL AND consumed IS NULL",
			cols),
		id.String())
	if err != nil && err != sql.ErrNoRows {
		rollback()
		return err
	}

	if err == nil {
		req.Key = stored.Key
		if err := req.AccountID.FromString(stored.AccountID); err == nil {
			account, err = b.get(t, req.AccountID)
			if err != nil && err != proto.ErrAccountNotFound {
				rollback()
				return err
			}
		}
	}

	if !req.VerifyMAC(mac) || account == nil {
		rollback()
		fmt.Printf("invalid mac or no account (%#v)\n", account)
		return proto.ErrInvalidConfirmationCode
	}

	sec, err := account.accountSecurity().ResetPassword(kms, password)
	if err != nil {
		rollback()
		fmt.Printf("reset password failed: %s\n", err)
		return err
	}

	_, err = t.Exec(
		"UPDATE account SET mac = $2, encrypted_user_key = $3 WHERE id = $1",
		account.ID().String(), sec.MAC, sec.UserKey.Ciphertext)
	if err != nil {
		rollback()
		fmt.Printf("update 1 failed: %s\n", err)
		return err
	}

	_, err = t.Exec("UPDATE password_reset_request SET consumed = NOW() where id = $1", id.String())
	if err != nil {
		rollback()
		fmt.Printf("update 2 failed: %s\n", err)
		return err
	}

	_, err = t.Exec(
		"UPDATE password_reset_request SET invalidated = NOW() where account_id = $1 AND id != $2",
		account.ID().String(), id)
	if err != nil {
		rollback()
		fmt.Printf("update 3 failed: %s\n", err)
		return err
	}

	if err := t.Commit(); err != nil {
		fmt.Printf("commit failed: %s\n", err)
		return err
	}

	return nil
}
