package psql

import (
	"database/sql"
	"strings"
	"time"

	"encoding/base64"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"

	"github.com/go-gorp/gorp"
)

type Agent struct {
	ID                 string
	IV                 []byte
	MAC                []byte
	EncryptedClientKey []byte         `db:"encrypted_client_key"`
	AccountID          sql.NullString `db:"account_id"`
}

type AgentTrackerBinding struct {
	*Backend
}

func (atb *AgentTrackerBinding) BanAgent(ctx scope.Context, agentID string, until time.Time) error {
	ban := &BannedAgent{
		AgentID: agentID,
		Created: time.Now(),
		Expires: gorp.NullTime{
			Time:  until,
			Valid: !until.IsZero(),
		},
	}

	if err := atb.Backend.DbMap.Insert(ban); err != nil {
		return err
	}

	bounceEvent := &proto.BounceEvent{Reason: "banned", AgentID: agentID}
	return atb.broadcast(ctx, nil, proto.BounceEventType, bounceEvent)
}

func (atb *AgentTrackerBinding) UnbanAgent(ctx scope.Context, agentID string) error {
	_, err := atb.Backend.DbMap.Exec(
		"DELETE FROM banned_agent WHERE agent_id = $1 AND room IS NULL", agentID)
	return err
}

func (atb *AgentTrackerBinding) Register(ctx scope.Context, agent *proto.Agent) error {
	row := &Agent{
		ID:  agent.IDString(),
		IV:  agent.IV,
		MAC: agent.MAC,
	}
	if agent.EncryptedClientKey != nil {
		row.EncryptedClientKey = agent.EncryptedClientKey.Ciphertext
	}

	if err := atb.Backend.DbMap.Insert(row); err != nil {
		if strings.HasPrefix(err.Error(), "pq: duplicate key value") {
			return proto.ErrAgentAlreadyExists
		}
		return err
	}

	return nil
}

func (atb *AgentTrackerBinding) Get(ctx scope.Context, agentID string) (*proto.Agent, error) {
	idBytes, err := base64.URLEncoding.DecodeString(agentID)
	if err != nil {
		return nil, proto.ErrAgentNotFound
	}

	row, err := atb.Backend.DbMap.Get(Agent{}, agentID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, proto.ErrAgentNotFound
	}

	agentRow := row.(*Agent)
	agent := &proto.Agent{
		ID:  idBytes,
		IV:  agentRow.IV,
		MAC: agentRow.MAC,
		EncryptedClientKey: &security.ManagedKey{
			IV:         agentRow.IV,
			Ciphertext: agentRow.EncryptedClientKey,
		},
	}
	return agent, nil
}

func (atb *AgentTrackerBinding) SetClientKey(
	ctx scope.Context, agentID string, accessKey, clientKey *security.ManagedKey) error {

	return notImpl
}
