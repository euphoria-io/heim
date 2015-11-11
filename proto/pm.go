package proto

import (
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
	"golang.org/x/crypto/poly1305"
)

type PMTracker interface {
	Initiate(ctx scope.Context, kms security.KMS, client *Client, receiver UserID) (snowflake.Snowflake, error)
	Room(ctx scope.Context, kms security.KMS, pmID snowflake.Snowflake, client *Client) (Room, *security.ManagedKey, error)
}

func NewPM(kms security.KMS, client *Client, receiver UserID) (*PM, *security.ManagedKey, error) {
	if client.Account == nil {
		return nil, nil, ErrAccessDenied
	}

	pmID, err := snowflake.New()
	if err != nil {
		return nil, nil, err
	}

	iv, err := kms.GenerateNonce(RoomMessageKeyType.BlockSize())
	if err != nil {
		return nil, nil, err
	}

	encryptedSystemKey, err := kms.GenerateEncryptedKey(RoomMessageKeyType, "pm", pmID.String())
	if err != nil {
		return nil, nil, err
	}

	pmKey := encryptedSystemKey.Clone()
	if err := kms.DecryptKey(&pmKey); err != nil {
		return nil, nil, err
	}

	userKey := client.Account.UserKey()
	if err := userKey.Decrypt(client.Authorization.ClientKey); err != nil {
		return nil, nil, err
	}

	encryptedInitiatorKey := encryptedSystemKey.Clone()
	if err := encryptedInitiatorKey.Encrypt(&userKey); err != nil {
		return nil, nil, err
	}

	var (
		mac [16]byte
		key [32]byte
	)
	copy(key[:], pmKey.Plaintext)
	poly1305.Sum(&mac, []byte(receiver), &key)

	pm := &PM{
		ID:                    pmID,
		Initiator:             client.Account.ID(),
		Receiver:              receiver,
		ReceiverMAC:           mac[:],
		IV:                    iv,
		EncryptedSystemKey:    encryptedSystemKey,
		EncryptedInitiatorKey: &encryptedInitiatorKey,
	}
	return pm, &pmKey, nil
}

type PM struct {
	ID                    snowflake.Snowflake
	Initiator             snowflake.Snowflake
	Receiver              UserID
	ReceiverMAC           []byte
	IV                    []byte
	EncryptedSystemKey    *security.ManagedKey
	EncryptedInitiatorKey *security.ManagedKey
	EncryptedReceiverKey  *security.ManagedKey
}

func (pm *PM) transmitToAccount(kms security.KMS, pmKey *security.ManagedKey, receiver Account) (*PM, error) {
	userKey := receiver.SystemKey()
	if err := kms.DecryptKey(&userKey); err != nil {
		return nil, err
	}

	encryptedReceiverKey := pmKey.Clone()
	if err := encryptedReceiverKey.Encrypt(&userKey); err != nil {
		return nil, err
	}

	pm.EncryptedReceiverKey = &encryptedReceiverKey
	return pm, nil
}

func InitiatePM(ctx scope.Context, b Backend, kms security.KMS, client *Client, receiver UserID) (*PM, error) {
	resolveAccount := func(accountIDStr string) (Account, error) {
		var accountID snowflake.Snowflake
		if err := accountID.FromString(accountIDStr); err != nil {
			return nil, err
		}
		return b.AccountManager().Get(ctx, accountID)
	}

	pm, pmKey, err := NewPM(kms, client, receiver)
	if err != nil {
		return nil, err
	}

	kind, id := receiver.Parse()
	switch kind {
	case "account":
		receiver, err := resolveAccount(id)
		if err != nil {
			return nil, err
		}
		return pm.transmitToAccount(kms, pmKey, receiver)
	case "agent", "bot":
		agent, err := b.AgentTracker().Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if agent.AccountID != "" {
			receiver, err := resolveAccount(agent.AccountID)
			if err != nil {
				return nil, err
			}
			return pm.transmitToAccount(kms, pmKey, receiver)
		}
		// We can't transmit the key to the agent until the agent joins the chat.
		return pm, nil
	default:
		return nil, ErrInvalidUserID
	}
}
