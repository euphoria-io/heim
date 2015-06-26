package mock

import (
	"sync"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type capabilities struct {
	sync.Mutex
	accountCapabilityIDs map[string]string
	accounts             map[string]proto.Account
	capabilities         map[string]security.Capability
}

func (cs *capabilities) Get(ctx scope.Context, cid string) (security.Capability, error) {
	c, ok := cs.capabilities[cid]
	if !ok {
		return nil, proto.ErrCapabilityNotFound
	}
	return c, nil
}

func (cs *capabilities) Save(ctx scope.Context, account proto.Account, c security.Capability) error {
	cs.Lock()
	defer cs.Unlock()

	if cs.capabilities == nil {
		cs.capabilities = map[string]security.Capability{}
		cs.accounts = map[string]proto.Account{}
	}

	cid := c.CapabilityID()
	cs.capabilities[cid] = c
	cs.accounts[cid] = account
	return nil
}

func (cs *capabilities) Remove(ctx scope.Context, cid string) error {
	cs.Lock()
	defer cs.Unlock()

	if _, ok := cs.capabilities[cid]; !ok {
		return proto.ErrCapabilityNotFound
	}
	delete(cs.capabilities, cid)
	delete(cs.accounts, cid)
	return nil
}
