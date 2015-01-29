package psql

type Capability struct {
	ID            string
	Nonce         []byte
	Digest        []byte
	EncryptedData []byte `id:"encrypted_data"`
}
