package proto

import (
	"fmt"

	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
	"golang.org/x/crypto/poly1305"
)

type PMTracker interface {
	Initiate(ctx scope.Context, kms security.KMS, room Room, client *Client, receiver UserID) (snowflake.Snowflake, error)
	Room(ctx scope.Context, kms security.KMS, pmID snowflake.Snowflake, client *Client) (Room, *security.ManagedKey, error)
}

func NewPM(kms security.KMS, client *Client, initiatorNick string, receiver UserID, receiverNick string) (
	*PM, *security.ManagedKey, error) {

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
		return nil, nil, fmt.Errorf("pm key decrypt: %s", err)
	}
	//pmKey.IV = iv

	userKey := client.Account.UserKey()
	if err := userKey.Decrypt(client.Authorization.ClientKey); err != nil {
		return nil, nil, fmt.Errorf("initiator account key decrypt: %s", err)
	}

	encryptedInitiatorKey := pmKey.Clone()
	encryptedInitiatorKey.IV = iv
	if err := encryptedInitiatorKey.Encrypt(&userKey); err != nil {
		return nil, nil, fmt.Errorf("initiator pm key encrypt: %s", err)
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
		InitiatorNick:         initiatorNick,
		Receiver:              receiver,
		ReceiverNick:          receiverNick,
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
	InitiatorNick         string
	Receiver              UserID
	ReceiverNick          string
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
	encryptedReceiverKey.IV = pm.IV
	if err := encryptedReceiverKey.Encrypt(&userKey); err != nil {
		return nil, err
	}

	pm.EncryptedReceiverKey = &encryptedReceiverKey
	return pm, nil
}

func (pm *PM) Access(ctx scope.Context, b Backend, kms security.KMS, client *Client) (*security.ManagedKey, bool, string, error) {
	if client.Authorization.ClientKey == nil {
		return nil, false, "", ErrAccessDenied
	}

	keyID := fmt.Sprintf("v1/pm:%s", pm.ID)

	if client.Account != nil && client.Account.ID() == pm.Initiator {
		userKey := client.Account.UserKey()
		if err := userKey.Decrypt(client.Authorization.ClientKey); err != nil {
			return nil, false, "", err
		}
		pmKey := pm.EncryptedInitiatorKey.Clone()
		if err := pmKey.Decrypt(&userKey); err != nil {
			return nil, false, "", err
		}
		client.Authorization.AddMessageKey(keyID, &pmKey)
		return &pmKey, false, pm.ReceiverNick, pm.verifyKey(&pmKey)
	}

	kind, _ := pm.Receiver.Parse()
	switch kind {
	case "account":
		if client.Account == nil {
			return nil, false, "", ErrAccessDenied
		}
		userKey := client.Account.UserKey()
		if err := userKey.Decrypt(client.Authorization.ClientKey); err != nil {
			return nil, false, "", err
		}
		pmKey := pm.EncryptedReceiverKey.Clone()
		if err := pmKey.Decrypt(&userKey); err != nil {
			return nil, false, "", err
		}
		client.Authorization.AddMessageKey(keyID, &pmKey)
		return &pmKey, false, pm.InitiatorNick, pm.verifyKey(&pmKey)
	case "agent", "bot":
		if client.Account != nil {
			pmKey, err := pm.upgradeToAccountReceiver(ctx, b, kms, client)
			if err != nil {
				return nil, false, "", err
			}
			client.Authorization.AddMessageKey(keyID, pmKey)
			return pmKey, true, pm.InitiatorNick, pm.verifyKey(pmKey)
		}
		if pm.EncryptedReceiverKey == nil {
			pmKey, err := pm.transmitToAgent(kms, client)
			if err != nil {
				return nil, false, "", err
			}
			client.Authorization.AddMessageKey(keyID, pmKey)
			return pmKey, true, pm.InitiatorNick, pm.verifyKey(pmKey)
		}
		pmKey := pm.EncryptedReceiverKey.Clone()
		if err := pmKey.Decrypt(client.Authorization.ClientKey); err != nil {
			return nil, false, "", err
		}
		client.Authorization.AddMessageKey(keyID, &pmKey)
		return &pmKey, false, pm.InitiatorNick, pm.verifyKey(&pmKey)
	default:
		return nil, false, "", ErrInvalidUserID
	}
}

func (pm *PM) verifyKey(pmKey *security.ManagedKey) error {
	var (
		mac [16]byte
		key [32]byte
	)
	copy(mac[:], pm.ReceiverMAC)
	copy(key[:], pmKey.Plaintext)
	if !poly1305.Verify(&mac, []byte(pm.Receiver), &key) {
		return ErrAccessDenied
	}
	return nil
}

func (pm *PM) upgradeToAccountReceiver(ctx scope.Context, b Backend, kms security.KMS, client *Client) (*security.ManagedKey, error) {
	// Verify that client and receiver agent share the same account.
	_, id := pm.Receiver.Parse()
	agent, err := b.AgentTracker().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if agent.AccountID != client.Account.ID().String() {
		return nil, ErrAccessDenied
	}

	// Unlock PM and verify Receiver.
	pmKey := pm.EncryptedSystemKey.Clone()
	if err := kms.DecryptKey(&pmKey); err != nil {
		return nil, err
	}
	if err := pm.verifyKey(&pmKey); err != nil {
		return nil, err
	}

	// Re-encrypt PM key for account.
	encryptedReceiverKey := pmKey.Clone()
	encryptedReceiverKey.IV = pm.IV
	if err := encryptedReceiverKey.Encrypt(client.Authorization.ClientKey); err != nil {
		return nil, err
	}
	pm.EncryptedReceiverKey = &encryptedReceiverKey
	pm.Receiver = UserID(fmt.Sprintf("account:%s", client.Account.ID()))
	return &pmKey, nil
}

func (pm *PM) transmitToAgent(kms security.KMS, client *Client) (*security.ManagedKey, error) {
	if client.UserID() != pm.Receiver {
		return nil, ErrAccessDenied
	}

	// Decrypt PM key
	pmKey := pm.EncryptedSystemKey.Clone()
	if err := kms.DecryptKey(&pmKey); err != nil {
		return nil, err
	}

	// Verify ReceiverMAC
	if err := pm.verifyKey(&pmKey); err != nil {
		return nil, err
	}

	// Encrypt PM key for agent
	encryptedPMKey := pmKey.Clone()
	encryptedPMKey.IV = pm.IV
	if err := encryptedPMKey.Encrypt(client.Authorization.ClientKey); err != nil {
		return nil, err
	}

	pm.EncryptedReceiverKey = &encryptedPMKey
	return &pmKey, nil
}

func InitiatePM(
	ctx scope.Context, b Backend, kms security.KMS, client *Client, initiatorNick string, receiver UserID,
	receiverNick string) (*PM, error) {

	resolveAccount := func(accountIDStr string) (Account, error) {
		var accountID snowflake.Snowflake
		if err := accountID.FromString(accountIDStr); err != nil {
			return nil, err
		}
		return b.AccountManager().Get(ctx, accountID)
	}

	pm, pmKey, err := NewPM(kms, client, initiatorNick, receiver, receiverNick)
	if err != nil {
		return nil, fmt.Errorf("new pm: %s", err)
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
