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
	"github.com/dominant-strategies/go-quai/crypto"
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

// KeyManager manages the creation, storage and loading of private keys
type KeyManager struct {
	storage keyStore // Storage backend, might be cleartext or encrypted
	keyDir  string
}

var _ KeyStoreManager = (*KeyManager)(nil)

// NewKeyManager creates a new KeyManager instance
func NewKeyManager(keyDir string) (*KeyManager, error) {
	// Ensure directory exists
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %v", err)
	}

	// TODO: Check encryption parameters here
	// Create keystore instance with standard scrypt parameters
	ks := NewKeyStore(keyDir, StandardScryptN, StandardScryptP)

	return &KeyManager{
		storage: ks,
		keyDir:  keyDir,
	}, nil
}

// CreateNewKey creates a new private key and stores it encrypted
func (k *KeyManager) CreateNewKey(location common.Location, protocol string) (common.Address, error) {
	// Get password with confirmation
	password, err := promptAndConfirmPassword("Enter password for new key: ")
	if err != nil {
		return common.Address{}, err
	}

	// Create new account
	account, err := k.NewAccount(password, location, protocol)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create new account: %v", err)
	}

	return account.Address, nil
}

// LoadFile loads a private key from a keystore file
func (k *KeyManager) LoadFile(keyFile string) (*Key, error) {
	// Read key file content
	keyjson, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}

	// Read password
	password, err := readPassword("Enter password to decrypt key: ")
	if err != nil {
		return nil, err
	}

	// Decrypt key
	key, err := DecryptKey(keyjson, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %v", err)
	}

	return key, nil
}

// LoadKey loads a private key from the keystore
func (k *KeyManager) LoadKey(address common.Address) (*Key, error) {
	// Read password
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

	// Get decrypted key
	key, err := k.GetKey(address, keyFile, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key: %v", err)
	}

	return key, nil
}

// readPassword securely reads a password
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line
	if err != nil {
		return "", fmt.Errorf("failed to read password: %v", err)
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
func (k *KeyManager) NewAccount(passphrase string, location common.Location, protocol string) (Account, error) {
	_, account, err := storeNewKey(k.storage, crand.Reader, passphrase, location, protocol)
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

// ImportPrivateKey imports a private key from a hex string and stores it encrypted
func (k *KeyManager) ImportPrivateKey() (common.Address, error) {
	// Read private key with hidden input
	privateKeyStr, err := readPassword("Enter private key (hex format): ")
	if err != nil {
		return common.Address{}, err
	}

	// Clean the private key string (remove 0x prefix if present)
	privateKeyStr = strings.TrimPrefix(strings.TrimSpace(privateKeyStr), "0x")

	// Convert hex string to ECDSA private key
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return common.Address{}, fmt.Errorf("invalid private key: %v", err)
	}

	// Generate random UUID for the key
	id, err := uuid.NewRandom()
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to generate UUID: %v", err)
	}

	// Create the Key struct
	key := &Key{
		Id:         id,
		Address:    PubkeyToAddressWithoutLocation(privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	// Get password with confirmation
	password, err := promptAndConfirmPassword("Enter password to encrypt key: ")
	if err != nil {
		return common.Address{}, err
	}

	// Create account URL
	account := Account{
		Address: key.Address,
		URL:     URL{Scheme: KeyStoreScheme, Path: k.storage.JoinPath(keyFileName(key.Address))},
	}

	// Store the key
	if err := k.storage.StoreKey(account.URL.Path, key, password); err != nil {
		return common.Address{}, fmt.Errorf("failed to store key: %v", err)
	}

	fmt.Printf("\nSuccessfully imported and encrypted key for address: %x\n", key.Address)
	return key.Address, nil
}

func PubkeyToAddressWithoutLocation(p ecdsa.PublicKey) common.Address {
	pubBytes := crypto.FromECDSAPub(&p)
	addressBytes := crypto.Keccak256(pubBytes[1:])[12:]
	lowerNib := addressBytes[0] & 0x0F        // Lower 4 bits
	upperNib := (addressBytes[0] & 0xF0) >> 4 // Upper 4 bits, shifted right
	location := common.Location{upperNib, lowerNib}
	return crypto.PubkeyToAddress(p, location)
}

// promptAndConfirmPassword prompts the user for a password and confirms it
func promptAndConfirmPassword(initialPrompt string) (string, error) {
	// Read password
	password, err := readPassword(initialPrompt)
	if err != nil {
		return "", err
	}

	// Confirm password
	confirmPass, err := readPassword("Confirm password: ")
	if err != nil {
		return "", err
	}

	if password != confirmPass {
		return "", fmt.Errorf("passwords do not match")
	}
	fmt.Println("Password match successful!")

	return password, nil
}
