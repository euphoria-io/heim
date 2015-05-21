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

type Authentication struct {
	Capability    security.Capability
	KeyID         string
	Key           *security.ManagedKey
	FailureReason string
}

func Authenticate(ctx scope.Context, room Room, cmd *AuthCommand) (*Authentication, error) {
	switch cmd.Type {
	case AuthPasscode:
		return authenticateWithPasscode(ctx, room, cmd.Passcode)
	default:
		return &Authentication{FailureReason: fmt.Sprintf("auth type not supported: %s", cmd.Type)}, nil
	}
}

func authenticateWithPasscode(ctx scope.Context, room Room, passcode string) (
	*Authentication, error) {

	mkey, err := room.MasterKey(ctx)
	if err != nil {
		return nil, err
	}

	if mkey == nil {
		return &Authentication{}, nil
	}

	holder := security.PasscodeCapabilityHolder([]byte(passcode), mkey.Nonce())
	subject := &roomCapabilitySubject{RoomKey: mkey}
	capabilityID, err := security.GetCapabilityID(holder, subject)
	if err != nil {
		return nil, err
	}

	capability, err := room.GetCapability(ctx, capabilityID)
	if err != nil {
		return nil, err
	}

	if capability == nil {
		return &Authentication{FailureReason: "passcode incorrect"}, nil
	}

	clientKey := security.KeyFromPasscode([]byte(passcode), mkey.Nonce(), security.AES128.KeySize())
	roomKey, err := decryptRoomKey(clientKey, capability)
	if err != nil {
		return nil, err
	}

	auth := &Authentication{
		Capability: capability,
		KeyID:      mkey.KeyID(),
		Key:        roomKey,
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
