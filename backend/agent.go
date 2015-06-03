package backend

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
	"github.com/gorilla/securecookie"
)

const (
	agentCookieName     = "a"
	agentCookieDuration = 365 * 24 * time.Hour
)

func newAgentCredentials(agent *proto.Agent, accessKey *security.ManagedKey) (*agentCredentials, error) {
	if accessKey.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	ac := &agentCredentials{
		ID:  agent.IDString(),
		Key: accessKey.Plaintext,
	}
	return ac, nil
}

type agentCredentials struct {
	ID  string `json:"i"`
	Key []byte `json:"k"`
}

func (ac *agentCredentials) Cookie(sc *securecookie.SecureCookie) (*http.Cookie, error) {
	encoded, err := json.Marshal(ac)
	if err != nil {
		return nil, err
	}

	secured, err := sc.Encode(agentCookieName, encoded)
	if err != nil {
		return nil, err
	}

	cookie := &http.Cookie{
		Name:     agentCookieName,
		Value:    secured,
		Path:     "/",
		Expires:  time.Now().Add(agentCookieDuration),
		HttpOnly: true,
	}
	return cookie, nil
}

func assignAgent(ctx scope.Context, s *Server) (*proto.Agent, *http.Cookie, *security.ManagedKey, error) {
	accessKey := &security.ManagedKey{
		KeyType:   proto.AgentKeyType,
		Plaintext: make([]byte, proto.AgentKeyType.KeySize()),
	}
	_, err := rand.Read(accessKey.Plaintext)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("agent generation error: %s", err)
	}

	var agentID []byte
	if s.agentIDGenerator != nil {
		agentID, err = s.agentIDGenerator()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("agent generation error: %s", err)
		}
	}

	agent, err := proto.NewAgent(agentID, accessKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("agent generation error: %s", err)
	}

	if err := s.b.AgentTracker().Register(ctx, agent); err != nil {
		return nil, nil, nil, fmt.Errorf("agent generation error: %s", err)
	}

	ac, err := newAgentCredentials(agent, accessKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("agent generation error: %s", err)
	}

	cookie, err := ac.Cookie(s.sc)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("agent generation error: %s", err)
	}

	return agent, cookie, nil, nil
}

func getAgent(
	ctx scope.Context, s *Server, r *http.Request) (
	*proto.Agent, *http.Cookie, *security.ManagedKey, error) {

	cookie, err := r.Cookie(agentCookieName)
	if err != nil {
		return assignAgent(ctx, s)
	}

	encoded := []byte{}
	if err := s.sc.Decode(agentCookieName, cookie.Value, &encoded); err != nil {
		return assignAgent(ctx, s)
	}

	ac := agentCredentials{}
	if err := json.Unmarshal(encoded, &ac); err != nil {
		return assignAgent(ctx, s)
	}

	agent, err := s.b.AgentTracker().Get(ctx, ac.ID)
	if err != nil {
		return assignAgent(ctx, s)
	}

	accessKey := &security.ManagedKey{
		KeyType:   proto.AgentKeyType,
		Plaintext: ac.Key,
	}
	clientKey, err := agent.Unlock(accessKey)
	if err != nil && err != proto.ErrClientKeyNotFound {
		return assignAgent(ctx, s)
	}

	cookie, err = ac.Cookie(s.sc)
	if err != nil {
		return nil, nil, nil, err
	}

	return agent, cookie, clientKey, nil
}
