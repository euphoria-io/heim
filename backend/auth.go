package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"heim/proto"
	"heim/proto/security"

	"golang.org/x/net/context"
)

type Authentication struct {
	Capability    security.Capability
	KeyID         string
	Key           *security.ManagedKey
	FailureReason string
}

func Authenticate(ctx context.Context, room proto.Room, cmd *proto.AuthCommand) (*Authentication, error) {
	switch cmd.Type {
	case proto.AuthPasscode:
		return authenticateWithPasscode(ctx, room, cmd.Passcode)
	default:
		return &Authentication{FailureReason: fmt.Sprintf("auth type not supported: %s", cmd.Type)}, nil
	}
}

func authenticateWithPasscode(ctx context.Context, room proto.Room, passcode string) (
	*Authentication, error) {

	mkey, err := room.MasterKey(ctx)
	if err != nil {
		return nil, err
	}

	if mkey == nil {
		return &Authentication{}, nil
	}

	capabilityID, err := security.GetCapabilityIDForPasscode(mkey.Nonce(), []byte(passcode))
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
