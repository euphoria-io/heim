package psql

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/gorp.v1"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

const OTPKeyType = security.AES128

type Account struct {
	ID                  string
	Name                string
	Email               string
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

type OTP struct {
	AccountID    string `db:"account_id"`
	IV           []byte
	EncryptedKey []byte `db:"encrypted_key"`
	Digest       []byte
	EncryptedURI []byte `db:"encrypted_uri"`
	Validated    bool
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

func (ab *AccountBinding) Name() string { return ab.Account.Name }

func (ab *AccountBinding) Email() (string, bool) {
	for _, pid := range ab.identities {
		if pid.Namespace() == "email" && ab.Account.Email == pid.ID() {
			return ab.Account.Email, pid.Verified()
		}
	}
	return ab.Account.Email, false
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

func (ab *AccountBinding) View(roomName string) *proto.AccountView {
	view := &proto.AccountView{
		ID:   ab.ID(),
		Name: ab.Name(),
	}
	return view
}

type AccountManagerBinding struct {
	*Backend
}

func (b *AccountManagerBinding) VerifyPersonalIdentity(ctx scope.Context, namespace, id string) error {
	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}

	checkResult := func(res sql.Result) error {
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return proto.ErrAccountNotFound
		}
		return nil
	}

	res, err := t.Exec(
		"UPDATE personal_identity SET verified = true WHERE namespace = $1 and id = $2",
		namespace, id)
	if err != nil {
		rollback(ctx, t)
		if err == sql.ErrNoRows {
			return proto.ErrAccountNotFound
		}
		return err
	}
	if err := checkResult(res); err != nil {
		rollback(ctx, t)
		return err
	}

	if namespace == "email" {
		// Look up ID of account that was verified.
		var row struct {
			ID string `db:"account_id"`
		}
		err = t.SelectOne(&row, "SELECT account_id FROM personal_identity WHERE namespace = 'email' AND id = $1", id)
		if err != nil {
			rollback(ctx, t)
			if err == sql.ErrNoRows {
				return proto.ErrAccountNotFound
			}
			return err
		}
		res, err = t.Exec("UPDATE account SET email = $2 WHERE id = $1", row.ID, id)
		if err != nil {
			rollback(ctx, t)
			if err == sql.ErrNoRows {
				return proto.ErrAccountNotFound
			}
			return err
		}
		if err := checkResult(res); err != nil {
			rollback(ctx, t)
			return err
		}
	}

	if err := t.Commit(); err != nil {
		return err
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
			logging.Logger(ctx).Printf("rollback error: %s", err)
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
			logging.Logger(ctx).Printf("rollback error: %s", err)
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
	if namespace == "email" {
		account.Email = id
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
	logging.Logger(ctx).Printf("registered new account %s for %s:%s", account.ID, namespace, id)

	ab := account.Bind(b.Backend)
	ab.identities = []proto.PersonalIdentity{&PersonalIdentityBinding{personalIdentity}}
	return ab, clientKey, nil
}

func (b *AccountManagerBinding) Resolve(ctx scope.Context, namespace, id string) (proto.Account, error) {
	t, err := b.DbMap.Begin()
	account, err := b.resolve(t, namespace, id)
	if err != nil {
		if rerr := t.Rollback(); rerr != nil {
			logging.Logger(ctx).Printf("rollback error: %s", err)
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
			logging.Logger(ctx).Printf("rollback error: %s", err)
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
			logging.Logger(ctx).Printf("rollback error: %s", err)
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
			logging.Logger(ctx).Printf("rollback error: %s", err)
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

func (b *AccountManagerBinding) resolvePasswordReset(
	db gorp.SqlExecutor, confirmation string) (*proto.PasswordResetRequest, *AccountBinding, error) {
	id, mac, err := proto.ParsePasswordResetConfirmation(confirmation)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	err = db.SelectOne(
		&stored,
		fmt.Sprintf(
			"SELECT %s FROM password_reset_request WHERE id = $1 AND expires > NOW() AND invalidated IS NULL AND consumed IS NULL",
			cols),
		id.String())
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	if err == nil {
		req.Key = stored.Key
		if err := req.AccountID.FromString(stored.AccountID); err == nil {
			account, err = b.get(db, req.AccountID)
			if err != nil && err != proto.ErrAccountNotFound {
				return nil, nil, err
			}
		}
	}

	if !req.VerifyMAC(mac) || account == nil {
		fmt.Printf("invalid mac or no account (%#v)\n", account)
		return nil, nil, proto.ErrInvalidConfirmationCode
	}

	return req, account, nil
}

func (b *AccountManagerBinding) GetPasswordResetAccount(ctx scope.Context, confirmation string) (proto.Account, error) {
	_, account, err := b.resolvePasswordReset(b.DbMap, confirmation)
	return account, err
}

func (b *AccountManagerBinding) ConfirmPasswordReset(
	ctx scope.Context, kms security.KMS, confirmation, password string) error {

	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}

	req, account, err := b.resolvePasswordReset(t, confirmation)
	if err != nil {
		rollback(ctx, t)
		return err
	}

	sec, err := account.accountSecurity().ResetPassword(kms, password)
	if err != nil {
		rollback(ctx, t)
		fmt.Printf("reset password failed: %s\n", err)
		return err
	}

	_, err = t.Exec(
		"UPDATE account SET mac = $2, encrypted_user_key = $3 WHERE id = $1",
		account.ID().String(), sec.MAC, sec.UserKey.Ciphertext)
	if err != nil {
		rollback(ctx, t)
		fmt.Printf("update 1 failed: %s\n", err)
		return err
	}

	_, err = t.Exec("UPDATE password_reset_request SET consumed = NOW() where id = $1", req.ID.String())
	if err != nil {
		rollback(ctx, t)
		fmt.Printf("update 2 failed: %s\n", err)
		return err
	}

	_, err = t.Exec(
		"UPDATE password_reset_request SET invalidated = NOW() where account_id = $1 AND id != $2",
		account.ID().String(), req.ID.String())
	if err != nil {
		rollback(ctx, t)
		fmt.Printf("update 3 failed: %s\n", err)
		return err
	}

	if err := t.Commit(); err != nil {
		fmt.Printf("commit failed: %s\n", err)
		return err
	}

	return nil
}

func (b *AccountManagerBinding) ChangeName(ctx scope.Context, accountID snowflake.Snowflake, name string) error {
	res, err := b.DbMap.Exec("UPDATE account SET name = $2 WHERE id = $1", accountID.String(), name)
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
	if n < 1 {
		return proto.ErrAccountNotFound
	}
	return nil
}

func (b *AccountManagerBinding) ChangeEmail(ctx scope.Context, accountID snowflake.Snowflake, email string) (bool, error) {
	t, err := b.DbMap.Begin()
	if err != nil {
		return false, err
	}

	account, err := b.get(t, accountID)
	if err != nil {
		rollback(ctx, t)
		return false, err
	}

	other, err := b.resolve(t, "email", email)
	if err != nil && err != proto.ErrAccountNotFound {
		rollback(ctx, t)
		return false, err
	}
	if err == nil && other.ID() != accountID {
		rollback(ctx, t)
		return false, proto.ErrPersonalIdentityInUse
	}

	for _, pid := range account.identities {
		if pid.Namespace() == "email" && pid.ID() == email {
			if pid.Verified() {
				res, err := t.Exec("UPDATE account SET email = $2 WHERE id = $1", accountID.String(), email)
				if err != nil {
					if err == sql.ErrNoRows {
						return false, proto.ErrAccountNotFound
					}
					rollback(ctx, t)
					return false, err
				}
				n, err := res.RowsAffected()
				if err != nil {
					rollback(ctx, t)
					return false, err
				}
				if n < 1 {
					rollback(ctx, t)
					return false, proto.ErrAccountNotFound
				}
				if err := t.Commit(); err != nil {
					return false, err
				}
				return true, nil
			}
			rollback(ctx, t)
			return false, nil
		}
	}

	pid := &PersonalIdentity{
		Namespace: "email",
		ID:        email,
		AccountID: accountID.String(),
	}
	if err := t.Insert(pid); err != nil {
		rollback(ctx, t)
		return false, err
	}

	if err := t.Commit(); err != nil {
		return false, err
	}

	return false, nil
}

func (b *AccountManagerBinding) getRawOTP(db gorp.SqlExecutor, accountID snowflake.Snowflake) (*OTP, error) {
	row, err := db.Get(OTP{}, accountID.String())
	if row == nil || err != nil {
		if row == nil || err == sql.ErrNoRows {
			return nil, proto.ErrOTPNotEnrolled
		}
		return nil, err
	}
	return row.(*OTP), nil
}

func (b *AccountManagerBinding) getOTP(db gorp.SqlExecutor, kms security.KMS, accountID snowflake.Snowflake) (*proto.OTP, error) {
	encryptedOTP, err := b.getRawOTP(db, accountID)
	if err != nil {
		return nil, err
	}

	key := security.ManagedKey{
		KeyType:      OTPKeyType,
		IV:           encryptedOTP.IV,
		Ciphertext:   encryptedOTP.EncryptedKey,
		ContextKey:   "account",
		ContextValue: accountID.String(),
	}
	if err := kms.DecryptKey(&key); err != nil {
		return nil, err
	}

	uriBytes, err := security.DecryptGCM(&key, encryptedOTP.IV, encryptedOTP.Digest, encryptedOTP.EncryptedURI, nil)
	if err != nil {
		return nil, err
	}

	otp := &proto.OTP{
		URI:       string(uriBytes),
		Validated: encryptedOTP.Validated,
	}
	return otp, nil
}

func (b *AccountManagerBinding) OTP(ctx scope.Context, kms security.KMS, accountID snowflake.Snowflake) (*proto.OTP, error) {
	return b.getOTP(b.DbMap, kms, accountID)
}

func (b *AccountManagerBinding) GenerateOTP(ctx scope.Context, heim *proto.Heim, kms security.KMS, account proto.Account) (*proto.OTP, error) {
	encryptedKey, err := kms.GenerateEncryptedKey(OTPKeyType, "account", account.ID().String())
	if err != nil {
		return nil, err
	}

	key := encryptedKey.Clone()
	if err := kms.DecryptKey(&key); err != nil {
		return nil, err
	}

	iv, err := kms.GenerateNonce(OTPKeyType.BlockSize())
	if err != nil {
		return nil, err
	}

	t, err := b.DbMap.Begin()
	if err != nil {
		return nil, err
	}

	rawOTP, err := b.getRawOTP(t, account.ID())
	if err != nil && err != proto.ErrOTPNotEnrolled {
		rollback(ctx, t)
		return nil, err
	}
	if err == nil {
		if rawOTP.Validated {
			rollback(ctx, t)
			return nil, proto.ErrOTPAlreadyEnrolled
		}
		row := &OTP{AccountID: account.ID().String()}
		if _, err := t.Delete(row); err != nil {
			rollback(ctx, t)
			return nil, err
		}
	}

	otp, err := heim.NewOTP(account)
	if err != nil {
		rollback(ctx, t)
		return nil, err
	}

	digest, encryptedURI, err := security.EncryptGCM(&key, iv, []byte(otp.URI), nil)
	if err != nil {
		rollback(ctx, t)
		return nil, err
	}

	row := &OTP{
		AccountID:    account.ID().String(),
		IV:           iv,
		EncryptedKey: encryptedKey.Ciphertext,
		Digest:       digest,
		EncryptedURI: encryptedURI,
	}
	if err := t.Insert(row); err != nil {
		// TODO: this could fail in the case of a race condition
		// by the time that matters we should be on postgres 9.5 and using a proper upsert
		rollback(ctx, t)
		return nil, err
	}

	if err := t.Commit(); err != nil {
		return nil, err
	}

	return otp, nil
}

func (b *AccountManagerBinding) ValidateOTP(ctx scope.Context, kms security.KMS, accountID snowflake.Snowflake, password string) error {
	t, err := b.DbMap.Begin()
	if err != nil {
		return err
	}

	otp, err := b.getOTP(t, kms, accountID)
	if err != nil {
		rollback(ctx, t)
		return err
	}

	if err := otp.Validate(password); err != nil {
		rollback(ctx, t)
		return err
	}

	if otp.Validated {
		rollback(ctx, t)
		return nil
	}

	res, err := t.Exec("UPDATE otp SET validated = true WHERE account_id = $1", accountID.String())
	if err != nil {
		rollback(ctx, t)
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		rollback(ctx, t)
		return err
	}
	if n != 1 {
		rollback(ctx, t)
		return fmt.Errorf("failed to mark otp enrollment as validated")
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}
