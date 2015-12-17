package psql

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"encoding/base64"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	"gopkg.in/gorp.v1"
)

type Agent struct {
	ID                 string
	IV                 []byte
	MAC                []byte
	EncryptedClientKey []byte         `db:"encrypted_client_key"`
	AccountID          sql.NullString `db:"account_id"`
	Created            time.Time
	Blessed            bool
	Bot                bool
}

type AgentTrackerBinding struct {
	*Backend
}

func (atb *AgentTrackerBinding) Register(ctx scope.Context, agent *proto.Agent) error {
	row := &Agent{
		ID:      agent.IDString(),
		IV:      agent.IV,
		MAC:     agent.MAC,
		Created: agent.Created,
		Bot:     agent.Bot,
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

	logging.Logger(ctx).Printf("registered agent %s", agent.IDString())
	return nil
}

func (atb *AgentTrackerBinding) getFromDB(agentID string, db gorp.SqlExecutor) (*proto.Agent, error) {
	idBytes, err := base64.URLEncoding.DecodeString(agentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent id %s: %s", agentID, err)
	}

	row, err := db.Get(Agent{}, agentID)
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
			KeyType:    proto.AgentKeyType,
			IV:         agentRow.IV,
			Ciphertext: agentRow.EncryptedClientKey,
		},
		AccountID: agentRow.AccountID.String,
		Created:   agentRow.Created,
		Blessed:   agentRow.Blessed,
		Bot:       agentRow.Bot,
	}
	return agent, nil
}

func (atb *AgentTrackerBinding) Get(ctx scope.Context, agentID string) (*proto.Agent, error) {
	return atb.getFromDB(agentID, atb.Backend.DbMap)
}

func (atb *AgentTrackerBinding) setClientKeyInDB(
	agentID, accountID string, keyBytes []byte, db gorp.SqlExecutor) error {

	_, err := db.Exec(
		"UPDATE agent SET account_id = $2, encrypted_client_key = $3 WHERE id = $1",
		agentID, accountID, keyBytes)
	if err != nil {
		return err
	}

	return nil
}

func (atb *AgentTrackerBinding) SetClientKey(
	ctx scope.Context, agentID string, accessKey *security.ManagedKey,
	accountID snowflake.Snowflake, clientKey *security.ManagedKey) error {

	t, err := atb.Backend.DbMap.Begin()
	if err != nil {
		return err
	}

	rollback := func() {
		if err := t.Rollback(); err != nil {
			logging.Logger(ctx).Printf("rollback error: %s", err)
		}
	}

	agent, err := atb.getFromDB(agentID, atb.Backend.DbMap)
	if err != nil {
		rollback()
		return err
	}

	if err := agent.SetClientKey(accessKey, clientKey); err != nil {
		rollback()
		return err
	}

	err = atb.setClientKeyInDB(
		agentID, accountID.String(), agent.EncryptedClientKey.Ciphertext, t)
	if err != nil {
		rollback()
		return err
	}

	if err := t.Commit(); err != nil {
		return err
	}

	return nil
}

func (atb *AgentTrackerBinding) ClearClientKey(ctx scope.Context, agentID string) error {
	resp, err := atb.Backend.DbMap.Exec(
		"UPDATE agent SET account_id = NULL, encrypted_client_key = '' WHERE id = $1",
		agentID)
	if err != nil {
		return err
	}

	n, err := resp.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return proto.ErrAgentNotFound
	}

	return nil
}
