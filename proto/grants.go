package proto

import (
	"encoding/json"
	"fmt"

	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type CapabilityTable interface {
	Get(ctx scope.Context, capabilityID string) (security.Capability, error)
	Save(ctx scope.Context, account Account, c security.Capability) error
	Remove(ctx scope.Context, capabilityID string) error
}

type AccountGrantable interface {
	GrantToAccount(
		ctx scope.Context, kms security.KMS, manager Account, managerClientKey *security.ManagedKey,
		target Account) error

	StaffGrantToAccount(ctx scope.Context, kms security.KMS, target Account) error

	RevokeFromAccount(ctx scope.Context, account Account) error

	AccountCapability(ctx scope.Context, account Account) (*security.PublicKeyCapability, error)
}

type PasscodeGrantable interface {
	GrantToPasscode(
		ctx scope.Context, manager Account, managerClientKey *security.ManagedKey, passcode string) error

	RevokeFromPasscode(ctx scope.Context, passcode string) error

	PasscodeCapability(ctx scope.Context, passcode string) (*security.SharedSecretCapability, error)
}

type GrantManager struct {
	Capabilities     CapabilityTable
	Managers         AccountGrantable
	KeyEncryptingKey *security.ManagedKey
	SubjectKeyPair   *security.ManagedKeyPair
	SubjectNonce     []byte
}

func (gs *GrantManager) unlockSubjectKeyPair(
	ctx scope.Context, manager Account, managerKeyPair *security.ManagedKeyPair) (
	*security.ManagedKeyPair, error) {

	// Get capability that unlocks gs.SubjectKeyPair.
	capability, err := gs.Managers.AccountCapability(ctx, manager)
	if err != nil {
		return nil, err
	}
	if capability == nil {
		return nil, ErrAccessDenied
	}

	secretJSON, err := capability.DecryptPayload(gs.SubjectKeyPair, managerKeyPair)
	if err != nil {
		return nil, fmt.Errorf("capability decryption error: %s", err)
	}

	// Unmarshal secretJSON into key-encrypting-key.
	managerKey := &security.ManagedKey{
		KeyType: security.AES128,
	}
	if err := json.Unmarshal(secretJSON, &managerKey.Plaintext); err != nil {
		return nil, fmt.Errorf("capability unmarshal error: %s", err)
	}

	// Unlock.
	subjectKeyPair := gs.SubjectKeyPair.Clone()
	if err := subjectKeyPair.Decrypt(managerKey); err != nil {
		return nil, fmt.Errorf("manager keypair unlock error: %s", err)
	}

	return &subjectKeyPair, nil
}

func (gs *GrantManager) Authority(
	ctx scope.Context, manager Account, managerKey *security.ManagedKey) (
	subjectKeyPair *security.ManagedKeyPair, public, private *json.RawMessage, err error) {

	managerKeyPair, err := manager.Unlock(managerKey)
	if err != nil {
		return nil, nil, nil, err
	}

	subjectKeyPair, err = gs.unlockSubjectKeyPair(ctx, manager, managerKeyPair)
	if err != nil {
		return nil, nil, nil, err
	}

	sourceCapability, err := gs.AccountCapability(ctx, manager)
	if err != nil {
		return nil, nil, nil, err
	}

	public = new(json.RawMessage)
	if publicBytes := sourceCapability.PublicPayload(); publicBytes != nil {
		*public = json.RawMessage(publicBytes)
	}

	private = new(json.RawMessage)
	privateBytes, err := sourceCapability.DecryptPayload(gs.SubjectKeyPair, managerKeyPair)
	if err != nil {
		return nil, nil, nil, err
	}
	*private = json.RawMessage(privateBytes)

	return
}

func (gs *GrantManager) GrantToAccount(
	ctx scope.Context, kms security.KMS, manager Account, managerKey *security.ManagedKey,
	target Account) error {

	subjectKeyPair, public, private, err := gs.Authority(ctx, manager, managerKey)
	if err != nil {
		return err
	}

	kp := target.KeyPair()
	c, err := security.GrantPublicKeyCapability(
		kms, gs.SubjectNonce, subjectKeyPair, &kp, public, private)
	if err != nil {
		return err
	}

	return gs.Capabilities.Save(ctx, target, c)
}

func (gs *GrantManager) StaffGrantToAccount(ctx scope.Context, kms security.KMS, target Account) error {
	keyEncryptingKey := gs.KeyEncryptingKey.Clone()
	if err := kms.DecryptKey(&keyEncryptingKey); err != nil {
		return fmt.Errorf("key-encrypting-key decrypt error: %s", err)
	}

	subjectKeyPair := gs.SubjectKeyPair.Clone()
	if err := subjectKeyPair.Decrypt(&keyEncryptingKey); err != nil {
		return err
	}

	// TODO: customize public/private payloads
	kp := target.KeyPair()
	c, err := security.GrantPublicKeyCapability(
		kms, gs.SubjectNonce, &subjectKeyPair, &kp, nil, keyEncryptingKey.Plaintext)
	if err != nil {
		return err
	}

	return gs.Capabilities.Save(ctx, target, c)
}

func (gs *GrantManager) RevokeFromAccount(ctx scope.Context, account Account) error {
	kp := account.KeyPair()
	cid := security.PublicKeyCapabilityID(gs.SubjectKeyPair, &kp, gs.SubjectNonce)
	return gs.Capabilities.Remove(ctx, cid)
}

func (gs *GrantManager) AccountCapability(
	ctx scope.Context, account Account) (*security.PublicKeyCapability, error) {

	kp := account.KeyPair()
	cid := security.PublicKeyCapabilityID(gs.SubjectKeyPair, &kp, gs.SubjectNonce)
	c, err := gs.Capabilities.Get(ctx, cid)
	if err != nil {
		if err == ErrCapabilityNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &security.PublicKeyCapability{Capability: c}, nil
}

func (gs *GrantManager) GrantToPasscode(
	ctx scope.Context, manager Account, managerKey *security.ManagedKey, passcode string) error {

	_, public, private, err := gs.Authority(ctx, manager, managerKey)
	if err != nil {
		return err
	}

	c, err := security.GrantSharedSecretCapability(
		security.KeyFromPasscode([]byte(passcode), gs.SubjectNonce, security.AES128),
		gs.SubjectNonce, public, private)
	if err != nil {
		return err
	}

	return gs.Capabilities.Save(ctx, nil, c)
}

func (gs *GrantManager) RevokeFromPasscode(ctx scope.Context, passcode string) error {
	cid, err := security.SharedSecretCapabilityID(
		security.KeyFromPasscode([]byte(passcode), gs.SubjectNonce, security.AES128),
		gs.SubjectNonce)
	if err != nil {
		return err
	}
	return gs.Capabilities.Remove(ctx, cid)
}

func (gs *GrantManager) PasscodeCapability(
	ctx scope.Context, passcode string) (*security.SharedSecretCapability, error) {

	cid, err := security.SharedSecretCapabilityID(
		security.KeyFromPasscode([]byte(passcode), gs.SubjectNonce, security.AES128),
		gs.SubjectNonce)
	if err != nil {
		return nil, err
	}

	c, err := gs.Capabilities.Get(ctx, cid)
	if err != nil {
		if err == ErrCapabilityNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &security.SharedSecretCapability{Capability: c}, nil
}
