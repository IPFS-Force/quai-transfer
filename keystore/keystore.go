package keystore

import (
	"path/filepath"
)

func NewKeyStore(keydir string, scryptN, scryptP int) keyStore {
	keydir, _ = filepath.Abs(keydir)
	ks := &keyStorePassphrase{keydir, scryptN, scryptP, false}
	return ks
}
