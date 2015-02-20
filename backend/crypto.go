package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"heim/proto"
	"heim/proto/security"
)

func decryptPayload(payload interface{}, roomKey *security.ManagedKey, capability security.Capability) (
	interface{}, error) {

	switch msg := payload.(type) {
	case proto.Message:
		if err := decryptMessage(&msg, roomKey, capability); err != nil {
			return nil, err
		}
		return msg, nil
	default:
		return msg, nil
	}
}

func encryptMessage(
	msg *proto.Message, roomKey *security.ManagedKey, capability security.Capability) error {

	if roomKey.Encrypted() {
		return security.ErrKeyMustBeDecrypted
	}

	// TODO: verify msg.ID makes sense as nonce
	digest, ciphertext, err := security.EncryptGCM(
		roomKey, []byte(msg.ID.String()), []byte(msg.Content), []byte(msg.Sender.ID))
	if err != nil {
		return fmt.Errorf("message encrypt: %s", err)
	}

	digestStr := base64.URLEncoding.EncodeToString(digest)
	cipherStr := base64.URLEncoding.EncodeToString(ciphertext)
	msg.Content = digestStr + "/" + cipherStr
	return nil
}

func decryptMessage(
	msg *proto.Message, roomKey *security.ManagedKey, capability security.Capability) error {

	if roomKey.Encrypted() {
		return security.ErrKeyMustBeDecrypted
	}

	parts := strings.Split(msg.Content, "/")
	if len(parts) != 2 {
		fmt.Printf("bad content: %s\n", msg.Content)
		return fmt.Errorf("message corrupted")
	}

	digest, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}

	ciphertext, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}

	plaintext, err := security.DecryptGCM(
		roomKey, []byte(msg.ID.String()), digest, ciphertext, []byte(msg.Sender.ID))
	if err != nil {
		return fmt.Errorf("message decrypt: %s", err)
	}

	msg.Content = string(plaintext)
	return nil
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
