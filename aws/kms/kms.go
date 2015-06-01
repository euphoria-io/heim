package kms

import (
	"fmt"

	"euphoria.io/heim/proto/security"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/awslabs/aws-sdk-go/service/kms"
)

func New(region, keyID string) (*KMS, error) {
	config := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		Region:      region,
	}
	kms := &KMS{
		kms:   kms.New(config),
		keyID: keyID,
	}
	return kms, nil
}

type KMS struct {
	kms   *kms.KMS
	keyID string
}

func (k *KMS) GenerateNonce(bytes int) ([]byte, error) {
	bytes64 := int64(bytes)
	resp, err := k.kms.GenerateRandom(&kms.GenerateRandomInput{NumberOfBytes: &bytes64})
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
		keySpec = "AES_128"
	case security.AES256:
		keySpec = "AES_256"
	default:
		return nil, fmt.Errorf("aws kms: key type %s not supported", keyType)
	}

	ctx := map[string]*string{ctxKey: &ctxVal}
	req := &kms.GenerateDataKeyWithoutPlaintextInput{
		KeyID:             &k.keyID,
		KeySpec:           &keySpec,
		EncryptionContext: &ctx,
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
	ctx := map[string]*string{key.ContextKey: &key.ContextValue}
	req := &kms.DecryptInput{
		CiphertextBlob:    key.Ciphertext,
		EncryptionContext: &ctx,
	}
	resp, err := k.kms.Decrypt(req)
	if err != nil {
		if apiErr, ok := err.(awserr.Error); ok && apiErr.Message() == "" {
			err = fmt.Errorf("%s", apiErr.Code())
		}
		return fmt.Errorf("aws kms: error decrypting data key: %s", err)
	}
	key.Plaintext = resp.Plaintext
	key.Ciphertext = nil
	return nil
}
