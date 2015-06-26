package psql

type MessageKey struct {
	ID           string
	EncryptedKey []byte `db:"encrypted_key"`
	IV           []byte
	Nonce        []byte
}

type Capability struct {
	ID                   string
	AccountID            string `db:"account_id"`
	NonceBytes           []byte `db:"nonce"`
	EncryptedPrivateData []byte `db:"encrypted_private_data"`
	PublicData           []byte `db:"public_data"`
}

func (c *Capability) CapabilityID() string     { return c.ID }
func (c *Capability) Nonce() []byte            { return c.NonceBytes }
func (c *Capability) PublicPayload() []byte    { return c.PublicData }
func (c *Capability) EncryptedPayload() []byte { return c.EncryptedPrivateData }
