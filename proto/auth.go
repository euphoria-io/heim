package proto

import "euphoria.io/heim/proto/security"

type AuthOption string

const (
	AuthPasscode = AuthOption("passcode")
)

type Authorization struct {
	ClientKey               *security.ManagedKey
	ManagerKeyEncryptingKey *security.ManagedKey
	ManagerKeyPair          *security.ManagedKeyPair
	MessageKeys             map[string]*security.ManagedKey
	CurrentMessageKeyID     string
}

func (a *Authorization) AddMessageKey(keyID string, key *security.ManagedKey) {
	if a.MessageKeys == nil {
		a.MessageKeys = map[string]*security.ManagedKey{keyID: key}
	} else {
		a.MessageKeys[keyID] = key
	}
}

type AuthorizationResult struct {
	Authorization
	FailureReason string
}

type Authentication struct {
	Capability     security.Capability
	KeyID          string
	Key            *security.ManagedKey
	AccountKeyPair *security.ManagedKeyPair
	FailureReason  string
}

func authorizationFailure(reason string) (*AuthorizationResult, error) {
	return &AuthorizationResult{FailureReason: reason}, nil
}
