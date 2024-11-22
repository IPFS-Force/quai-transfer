package keystore

import (
	"path/filepath"

	"github.com/dominant-strategies/go-quai/common"
)

type keyStore interface {
	// GetKey Loads and decrypts the key from disk.
	GetKey(addr common.Address, filename string, auth string) (*Key, error)
	// StoreKey Writes and encrypts the key.
	StoreKey(filename string, k *Key, auth string) error
	// JoinPath Joins filename with the key directory unless it is already absolute.
	JoinPath(filename string) string
}

func NewKeyStore(keydir string, scryptN, scryptP int) keyStore {
	keydir, _ = filepath.Abs(keydir)
	ks := &keyStorePassphrase{keydir, scryptN, scryptP, false}
	return ks
}
