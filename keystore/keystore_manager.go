package keystore

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/google/uuid"
	"golang.org/x/term"
)

var (
	ErrLocked  = NewAuthNeededError("password or unlock")
	ErrNoMatch = errors.New("no key for given address or file")
	ErrDecrypt = errors.New("could not decrypt key with given password")

	// ErrAccountAlreadyExists is returned if an account attempted to import is
	// already present in the keystore.
	ErrAccountAlreadyExists = errors.New("account already exists")
)

// KeyStoreScheme is the protocol scheme prefixing account and wallet URLs.
const KeyStoreScheme = "keystore"

// KeyManager 管理私钥的创建、存储和加载
type KeyManager struct {
	storage keyStore // Storage backend, might be cleartext or encrypted
	keyDir  string
}

type Key struct {
	Id uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address common.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

// NewKeyManager 创建一个新的KeyManager实例
func NewKeyManager(keyDir string) (*KeyManager, error) {
	// 确保目录存在
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %v", err)
	}

	// 创建keystore实例，使用标准的scrypt参数
	ks := NewKeyStore(keyDir, StandardScryptN, StandardScryptP)

	return &KeyManager{
		storage: ks,
		keyDir:  keyDir,
	}, nil
}

// CreateNewKey 创建新的私钥并加密存储
func (k *KeyManager) CreateNewKey(location common.Location) (common.Address, error) {
	// 读取密码
	password, err := readPassword("Enter password for new key: ")
	if err != nil {
		return common.Address{}, err
	}

	// 确认密码
	confirmPass, err := readPassword("Confirm password: ")
	if err != nil {
		return common.Address{}, err
	}

	if password != confirmPass {
		return common.Address{}, fmt.Errorf("passwords do not match")
	}
	fmt.Println("Password match successful!")

	// 创建新账户
	account, err := k.NewAccount(password, location)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create new account: %v", err)
	}

	return account.Address, nil
}

// LoadFile 从keystore文件加载私钥
func (k *KeyManager) LoadFile(keyFile string) (*Key, error) {
	// 读取密码
	password, err := readPassword("Enter password to decrypt key: ")
	if err != nil {
		return nil, err
	}

	// 读取文件内容
	keyjson, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}

	// 解密key
	key, err := DecryptKey(keyjson, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %v", err)
	}

	return key, nil
}

// LoadKey 从keystore加载私钥
func (k *KeyManager) LoadKey(address common.Address) (*Key, error) {
	// 读取密码
	password, err := readPassword("Enter password to decrypt key: ")
	if err != nil {
		return nil, err
	}

	// Find key file with matching address prefix
	files, err := os.ReadDir(k.keyDir)
	if err != nil {
		return nil, err
	}
	addrHex := hex.EncodeToString(address.Bytes()[:])
	var keyFile string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), addrHex) {
			keyFile = filepath.Join(k.keyDir, file.Name())
			break
		}
	}
	if keyFile == "" {
		return nil, fmt.Errorf("key file not found for address %x", address)
	}

	// 获取解密后的key
	key, err := k.GetKey(address, keyFile, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %v", err)
	}

	return key, nil
}

// readPassword 安全地读取密码
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // 换行
	if err != nil {
		return "", fmt.Errorf("读取密码失败: %v", err)
	}
	return string(bytePassword), nil
}

func (k *KeyManager) GetKey(addr common.Address, filename, auth string) (*Key, error) {
	// Load the key from the keystore and decrypt its contents
	keyjson, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	key, err := DecryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if !key.Address.Equal(addr) {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, addr)
	}
	return key, nil
}

// Export exports as a JSON key, encrypted with newPassphrase.
func (k *KeyManager) Export(a Account, passphrase, newPassphrase string) (keyJSON []byte, err error) {
	key, err := k.getDecryptedKey(a, passphrase)
	if err != nil {
		return nil, err
	}
	var N, P int
	if store, ok := k.storage.(*keyStorePassphrase); ok {
		N, P = store.scryptN, store.scryptP
	} else {
		N, P = StandardScryptN, StandardScryptP
	}
	return EncryptKey(key, newPassphrase, N, P)
}

func (k *KeyManager) getDecryptedKey(a Account, auth string) (*Key, error) {
	key, err := k.GetKey(a.Address, a.URL.Path, auth)
	return key, err
}

// zeroKey zeroes a private key in memory.
func zeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	clear(b)
}

// NewAccount generates a new key and stores it into the key directory,
// encrypting it with the passphrase.
func (k *KeyManager) NewAccount(passphrase string, location common.Location) (Account, error) {
	_, account, err := storeNewKey(k.storage, crand.Reader, passphrase, location)
	if err != nil {
		return Account{}, err
	}
	return account, nil
}

// NewAuthNeededError creates a new authentication error with the extra details
// about the needed fields set.
func NewAuthNeededError(needed string) error {
	return &AuthNeededError{
		Needed: needed,
	}
}

// Error implements the standard error interface.
func (err *AuthNeededError) Error() string {
	return fmt.Sprintf("authentication needed: %s", err.Needed)
}

// AuthNeededError is returned by backends for signing requests where the user
// is required to provide further authentication before signing can succeed.
//
// This usually means either that a password needs to be supplied, or perhaps a
// one time PIN code displayed by some hardware device.
type AuthNeededError struct {
	Needed string // Extra authentication the user needs to provide
}
