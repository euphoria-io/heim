package psql

import (
	"encoding/base64"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
)

type Account struct {
	ID                  string
	Nonce               []byte
	MAC                 []byte
	EncryptedSystemKey  []byte `db:"encrypted_system_key"`
	EncryptedUserKey    []byte `db:"encrypted_user_key"`
	EncryptedPrivateKey []byte `db:"encrypted_private_key"`
	PublicKey           []byte `db:"public_key"`
}

func (a *Account) Bind(b *Backend) *AccountBinding {
	return &AccountBinding{
		Backend: b,
		Account: a,
	}
}

type PersonalIdentity struct {
	Namespace string
	ID        string
	AccountID string `db:"account_id"`
}

type AccountBinding struct {
	*Backend
	*Account
}

func (ab *AccountBinding) ID() snowflake.Snowflake {
	var id snowflake.Snowflake
	_ = id.FromString(ab.Account.ID)
	return id
}

func (ab *AccountBinding) KeyFromPassword(password string) *security.ManagedKey {
	return security.KeyFromPasscode([]byte(password), ab.Account.Nonce, security.AES256)
}

func (ab *AccountBinding) KeyPair() security.ManagedKeyPair {
	iv := make([]byte, security.AES256.BlockSize())
	copy(iv, ab.Account.Nonce)

	return security.ManagedKeyPair{
		KeyPairType:         security.Curve25519,
		IV:                  iv,
		EncryptedPrivateKey: ab.Account.EncryptedPrivateKey,
		PublicKey:           ab.Account.PublicKey,
	}
}

func (ab *AccountBinding) Unlock(clientKey *security.ManagedKey) (*security.ManagedKeyPair, error) {
	iv := make([]byte, security.AES256.BlockSize())
	copy(iv, ab.Account.Nonce)

	sec := &proto.AccountSecurity{
		Nonce: ab.Account.Nonce,
		MAC:   ab.Account.MAC,
		SystemKey: security.ManagedKey{
			KeyType:      security.AES256,
			Ciphertext:   ab.Account.EncryptedSystemKey,
			ContextKey:   "nonce",
			ContextValue: base64.URLEncoding.EncodeToString(ab.Account.Nonce),
		},
		UserKey: security.ManagedKey{
			KeyType:    security.AES256,
			IV:         iv,
			Ciphertext: ab.Account.EncryptedUserKey,
		},
		KeyPair: security.ManagedKeyPair{
			KeyPairType:         security.Curve25519,
			IV:                  iv,
			EncryptedPrivateKey: ab.Account.EncryptedPrivateKey,
			PublicKey:           ab.Account.PublicKey,
		},
	}
	return sec.Unlock(clientKey)
}
