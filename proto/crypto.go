package proto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"euphoria.io/heim/proto/security"
)

func DecryptPayload(payload interface{}, auth map[string]*Authentication) (interface{}, error) {
	switch msg := payload.(type) {
	case Message:
		return DecryptMessage(msg, auth)
	case SendReply:
		dm, err := DecryptMessage(Message(msg), auth)
		if err != nil {
			return nil, err
		}
		return SendReply(dm), nil
	case *SendEvent:
		dm, err := DecryptMessage(Message(*msg), auth)
		if err != nil {
			return nil, err
		}
		return (*SendEvent)(&dm), nil
	case LogReply:
		for i, entry := range msg.Log {
			dm, err := DecryptPayload(entry, auth)
			if err != nil {
				return nil, err
			}
			msg.Log[i] = dm.(Message)
		}
		return msg, nil
	default:
		return msg, nil
	}
}

func EncryptMessage(msg *Message, keyID string, key *security.ManagedKey) error {
	if key == nil {
		return security.ErrInvalidKey
	}
	if key.Encrypted() {
		return security.ErrKeyMustBeDecrypted
	}

	payload := &Message{
		Sender:  msg.Sender,
		Content: msg.Content,
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// TODO: verify msg.ID makes sense as nonce
	nonce := []byte(msg.ID.String())
	data := []byte(msg.Sender.ID)

	digest, ciphertext, err := security.EncryptGCM(key, nonce, plaintext, data)
	if err != nil {
		return fmt.Errorf("message encrypt: %s", err)
	}

	digestStr := base64.URLEncoding.EncodeToString(digest)
	cipherStr := base64.URLEncoding.EncodeToString(ciphertext)

	msg.Sender = &IdentityView{ID: msg.Sender.ID}
	msg.Content = digestStr + "/" + cipherStr
	msg.EncryptionKeyID = "v1/" + keyID
	return nil
}

func DecryptMessage(msg Message, auths map[string]*Authentication) (Message, error) {
	if msg.EncryptionKeyID == "" {
		return msg, nil
	}

	keyVersion := 0
	keyID := msg.EncryptionKeyID
	if strings.HasPrefix(keyID, "v1") {
		keyVersion = 1
		keyID = keyID[3:]
	}

	auth, ok := auths[keyID]
	if !ok {
		return msg, nil
	}

	if auth.Key.Encrypted() {
		return msg, security.ErrKeyMustBeDecrypted
	}

	parts := strings.Split(msg.Content, "/")
	if len(parts) != 2 {
		fmt.Printf("bad content: %s\n", msg.Content)
		return msg, fmt.Errorf("message corrupted")
	}

	digest, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return msg, err
	}

	ciphertext, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return msg, err
	}

	plaintext, err := security.DecryptGCM(
		auth.Key, []byte(msg.ID.String()), digest, ciphertext, []byte(msg.Sender.ID))
	if err != nil {
		return msg, fmt.Errorf("message decrypt: %s", err)
	}

	switch keyVersion {
	case 0:
		// unversioned keys briefly existed when private rooms first rolled out
		msg.Content = string(plaintext)
	case 1:
		// v1 keys embed an encrypted partial message, hiding the sender and content
		var payload Message
		if err := json.Unmarshal(plaintext, &payload); err != nil {
			return msg, fmt.Errorf("message corrupted: %s", err)
		}
		msg.Sender = payload.Sender
		msg.Content = payload.Content
	}
	return msg, nil
}
