package psql

type MasterKey struct {
	ID           string
	EncryptedKey []byte `db:"encrypted_key"`
	IV           []byte
	Nonce        []byte
}

type Capability struct {
	ID                   string
	Kind                 string
	Nonce                []byte
	EncryptedPrivateData []byte `db:"encrypted_private_data"`
	PublicData           []byte `db:"public_data"`
}
