package proto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type AuthOption string

const (
	AuthPasscode = AuthOption("passcode")
)

type Authorization struct {
	ClientKey      *security.ManagedKey
	HasManagerKey  bool
	ManagerKeyPair *security.ManagedKeyPair
	MessageKeys    map[string]*security.ManagedKey
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

func Authenticate(
	ctx scope.Context, backend Backend, room Room, cmd *AuthCommand) (*AuthorizationResult, error) {

	switch cmd.Type {
	case AuthPasscode:
		return authenticateWithPasscode(ctx, room, cmd.Passcode)
	default:
		return authorizationFailure(fmt.Sprintf("auth type not supported: %s", cmd.Type))
	}
}

func authenticateWithPasscode(ctx scope.Context, room Room, passcode string) (
	*AuthorizationResult, error) {

	mkey, err := room.MessageKey(ctx)
	if err != nil {
		return nil, err
	}

	if mkey == nil {
		return &AuthorizationResult{}, nil
	}

	holderKey := security.KeyFromPasscode([]byte(passcode), mkey.Nonce(), security.AES128)

	capabilityID, err := security.SharedSecretCapabilityID(holderKey, mkey.Nonce())
	if err != nil {
		return nil, err
	}

	capability, err := room.GetCapability(ctx, capabilityID)
	if err != nil {
		return nil, err
	}

	if capability == nil {
		return authorizationFailure("passcode incorrect")
	}

	roomKey, err := decryptRoomKey(holderKey, capability)
	if err != nil {
		return nil, err
	}

	// TODO: load and return all keys
	auth := &AuthorizationResult{
		Authorization: Authorization{
			MessageKeys: map[string]*security.ManagedKey{mkey.KeyID(): roomKey},
		},
	}
	return auth, nil
}

func decryptRoomKey(clientKey *security.ManagedKey, capability security.Capability) (
	*security.ManagedKey, error) {

	if clientKey.Encrypted() {
		return nil, security.ErrKeyMustBeDecrypted
	}

	iv, err := base64.URLEncoding.DecodeString(capability.CapabilityID())
	if err != nil {
		return nil, err
	}

	roomKeyJSON := capability.EncryptedPayload()
	if err := clientKey.BlockCrypt(iv, clientKey.Plaintext, roomKeyJSON, false); err != nil {
		return nil, err
	}

	roomKey := &security.ManagedKey{
		KeyType: security.AES128,
	}
	if err := json.Unmarshal(clientKey.Unpad(roomKeyJSON), &roomKey.Plaintext); err != nil {
		return nil, err
	}
	return roomKey, nil
}
