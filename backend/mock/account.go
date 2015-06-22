package mock

import (
	"fmt"

	"euphoria.io/heim/backend"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
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
	id       snowflake.Snowflake
	sec      proto.AccountSecurity
	staffKMS security.KMS
}

func (a *memAccount) ID() snowflake.Snowflake { return a.id }

func (a *memAccount) KeyFromPassword(password string) *security.ManagedKey {
	return security.KeyFromPasscode([]byte(password), a.sec.Nonce, a.sec.UserKey.KeyType)
}

func (a *memAccount) KeyPair() security.ManagedKeyPair { return a.sec.KeyPair.Clone() }

func (a *memAccount) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	return a.sec.Unlock(clientKey)
}

func (a *memAccount) IsStaff() bool { return a.staffKMS != nil }

func (a *memAccount) UnlockStaffKMS(key *security.ManagedKey) (security.KMS, error) {
	// TODO: verify key
	if a.staffKMS == nil {
		return nil, proto.ErrAccessDenied
	}
	return a.staffKMS, nil
}

type accountManager struct {
	b *TestBackend
}

func (m *accountManager) Register(
	ctx scope.Context, kms security.KMS, namespace, id, password string,
	agentID string, agentKey *security.ManagedKey) (
	proto.Account, *security.ManagedKey, error) {

	m.b.Lock()
	defer m.b.Unlock()

	key := fmt.Sprintf("%s:%s", namespace, id)
	if _, ok := m.b.accountIDs[key]; ok {
		return nil, nil, proto.ErrPersonalIdentityInUse
	}

	account, clientKey, err := NewAccount(kms, password)
	if err != nil {
		return nil, nil, err
	}

	if m.b.accounts == nil {
		m.b.accounts = map[string]proto.Account{account.ID().String(): account}
	} else {
		m.b.accounts[account.ID().String()] = account
	}

	if m.b.accountIDs == nil {
		m.b.accountIDs = map[string]string{key: account.ID().String()}
	} else {
		m.b.accountIDs[key] = account.ID().String()
	}

	agent, err := m.b.AgentTracker().Get(ctx, agentID)
	if err != nil {
		backend.Logger(ctx).Printf(
			"error locating agent %s for new account %s:%s: %s", agentID, namespace, id, err)
	} else {
		if err := agent.SetClientKey(agentKey, clientKey); err != nil {
			backend.Logger(ctx).Printf(
				"error associating agent %s with new account %s:%s: %s", agentID, namespace, id, err)
		}
		agent.AccountID = account.ID().String()
	}

	return account, clientKey, nil
}

func (m *accountManager) Resolve(ctx scope.Context, namespace, id string) (proto.Account, error) {
	m.b.Lock()
	defer m.b.Unlock()

	key := fmt.Sprintf("%s:%s", namespace, id)
	accountID, ok := m.b.accountIDs[key]
	if !ok {
		return nil, proto.ErrAccountNotFound
	}
	return m.b.accounts[accountID], nil
}

func (m *accountManager) Get(ctx scope.Context, id snowflake.Snowflake) (proto.Account, error) {
	m.b.Lock()
	defer m.b.Unlock()

	account, ok := m.b.accounts[id.String()]
	if !ok {
		return nil, proto.ErrAccountNotFound
	}
	return account, nil
}

func (m *accountManager) setStaffKMS(accountID snowflake.Snowflake, kms security.KMS) error {
	m.b.Lock()
	defer m.b.Unlock()

	account, ok := m.b.accounts[accountID.String()]
	if !ok {
		return proto.ErrAccountNotFound
	}
	memAcc := account.(*memAccount)
	memAcc.staffKMS = kms
	return nil
}

func (m *accountManager) GrantStaff(
	ctx scope.Context, accountID snowflake.Snowflake, kmsCred security.KMSCredential) error {

	return m.setStaffKMS(accountID, kmsCred.KMS())
}

func (m *accountManager) RevokeStaff(ctx scope.Context, accountID snowflake.Snowflake) error {
	return m.setStaffKMS(accountID, nil)
}
