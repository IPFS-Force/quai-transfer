package wallet

import (
	"context"
	"math/big"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/core/types"
	wtypes "quai-transfer/types"
)

// WalletFunc defines the core functionality of a wallet
type WalletFunc interface {
	// Basic wallet operations
	GetBalance(ctx context.Context) (*big.Int, error)
	GetAddress() common.Address
	GetLocation() common.Location
	GetChainID(ctx context.Context) (*big.Int, error)
	Close()

	// Transaction operations
	ProcessEntry(ctx context.Context, entry *wtypes.TransferEntry) error
	CreateTransaction(ctx context.Context, entry *wtypes.TransferEntry) (*types.Transaction, error)
	BroadcastTransaction(ctx context.Context, tx *types.Transaction) error
	WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)

	// Address validation
	IsValidAddress(address string) bool
	IsValidQuaiAddress(address string) bool
	IsValidQiAddress(address string) bool

	// Transaction utilities
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
}

// IsInQuaiLedgerScope checks if an address is in the Quai ledger scope
func IsInQuaiLedgerScope(address string) bool {
	// The first bit of the second byte is not set if the address is in the Quai ledger
	return address[1] <= 127
}

// IsInQiLedgerScope checks if an address is in the Qi ledger scope
func IsInQiLedgerScope(address string) bool {
	// The first bit of the second byte is set if the address is in the Qi ledger
	return address[1] > 127
}

// ToWei converts an Ethereum value in val (as a string) to wei (as *big.Int)
func ToWei(v string) (*big.Int, bool) {
	value, ok := new(big.Float).SetString(v)
	if !ok {
		return nil, false // Could not parse ETH value
	}

	multiplier := new(big.Float).SetInt(big.NewInt(1e18))

	// Multiply the value by the conversion factor to get wei
	value.Mul(value, multiplier)

	wei := new(big.Int)
	value.Int(wei) // Extracts the integer part of the big.Float

	return wei, true
}
