package keystore

import "github.com/dominant-strategies/go-quai/common"

type KeyCreator interface {
	CreateNewKey(location common.Location, protocol string) (common.Address, error)
	NewAccount(passphrase string, location common.Location, protocol string) (Account, error)
	ImportPrivateKey() (common.Address, error)
}

type KeyLoader interface {
	LoadKey(address common.Address) (*Key, error)
	LoadFile(keyFile string) (*Key, error)
	GetKey(addr common.Address, filename, auth string) (*Key, error)
}

type KeyExporter interface {
	Export(a Account, passphrase, newPassphrase string) ([]byte, error)
}

type KeyStoreManager interface {
	KeyCreator
	KeyLoader
	KeyExporter
}
