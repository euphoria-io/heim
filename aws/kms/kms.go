package kms

import (
	"fmt"

	"euphoria.io/heim/proto/security"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/kms"
)

func New(region, keyID string) (*KMS, error) {
	creds, err := aws.EnvCreds()
	if err != nil {
		return nil, fmt.Errorf("aws kms: %s", err)
	}

	kms := &KMS{
		kms:   kms.New(creds, region, nil),
		keyID: keyID,
	}
	return kms, nil
}

type KMS struct {
	kms   *kms.KMS
	keyID string
}

func (k *KMS) GenerateNonce(bytes int) ([]byte, error) {
	resp, err := k.kms.GenerateRandom(&kms.GenerateRandomRequest{NumberOfBytes: &bytes})
	if err != nil {
		return nil, fmt.Errorf("aws kms: error generating nonce of %d bytes: %s", bytes, err)
	}
	return resp.Plaintext, nil
}

func (k *KMS) GenerateEncryptedKey(keyType security.KeyType, ctxKey, ctxVal string) (
	*security.ManagedKey, error) {

	var keySpec string
	switch keyType {
	case security.AES128:
		keySpec = kms.DataKeySpecAES128
	case security.AES256:
		keySpec = kms.DataKeySpecAES256
	default:
		return nil, fmt.Errorf("aws kms: key type %s not supported", keyType)
	}

	req := &kms.GenerateDataKeyWithoutPlaintextRequest{
		KeyID:             &k.keyID,
		KeySpec:           &keySpec,
		EncryptionContext: map[string]string{ctxKey: ctxVal},
	}

	resp, err := k.kms.GenerateDataKeyWithoutPlaintext(req)
	if err != nil {
		return nil, fmt.Errorf("aws kms: error generating data key of type %s: %s", keyType, err)
	}

	mkey := &security.ManagedKey{
		Ciphertext:   resp.CiphertextBlob,
		ContextKey:   ctxKey,
		ContextValue: ctxVal,
	}
	return mkey, nil
}

func (k *KMS) DecryptKey(key *security.ManagedKey) error {
	if !key.Encrypted() {
		return fmt.Errorf("aws kms: key is already decrypted")
	}
	req := &kms.DecryptRequest{
		CiphertextBlob:    key.Ciphertext,
		EncryptionContext: map[string]string{key.ContextKey: key.ContextValue},
	}
	resp, err := k.kms.Decrypt(req)
	if err != nil {
		if apiErr, ok := err.(aws.APIError); ok && apiErr.Message == "" {
			apiErr.Message = apiErr.Type
			err = fmt.Errorf("%s", apiErr.Type)
		}
		return fmt.Errorf("aws kms: error decrypting data key: %s", err)
	}
	key.Plaintext = resp.Plaintext
	key.Ciphertext = nil
	return nil
}
