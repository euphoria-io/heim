package mock

import (
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type agentTracker struct {
	b *TestBackend
}

func (t *agentTracker) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	return t.b.BanAgent(ctx, agentID, until)
}

func (t *agentTracker) UnbanAgent(ctx scope.Context, agentID string) error {
	return t.b.UnbanAgent(ctx, agentID)
}

func (t *agentTracker) Register(ctx scope.Context, agent *proto.Agent) error {
	t.b.Lock()
	defer t.b.Unlock()

	if _, ok := t.b.agents[agent.IDString()]; ok {
		return proto.ErrAgentAlreadyExists
	}
	if t.b.agents == nil {
		t.b.agents = map[string]*proto.Agent{agent.IDString(): agent}
	} else {
		t.b.agents[agent.IDString()] = agent
	}
	return nil
}

func (t *agentTracker) Get(ctx scope.Context, agentID string) (*proto.Agent, error) {
	agent, ok := t.b.agents[agentID]
	if !ok {
		return nil, proto.ErrAgentNotFound
	}
	return agent, nil
}

func (t *agentTracker) SetClientKey(
	ctx scope.Context, agentID string, accessKey, clientKey *security.ManagedKey) error {

	t.b.Lock()
	defer t.b.Unlock()

	agent, err := t.Get(ctx, agentID)
	if err != nil {
		return err
	}
	return agent.SetClientKey(accessKey, clientKey)
}
