package wallet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/dominant-strategies/go-quai/crypto"
	"github.com/dominant-strategies/go-quai/quaiclient/ethclient"
	"quai-transfer/keystore"
)

// Network represents different Quai networks
type Network string

const (
	Colosseum  Network = "colosseum"
	Garden     Network = "garden"
	Orchard    Network = "orchard"
	Lighthouse Network = "lighthouse"
	Local      Network = "local"
)

// NetworkConfig holds network specific configuration
type NetworkConfig struct {
	ChainID *big.Int
	RPCURLs map[string]string
}

// locationToString converts a Location to a string key
func locationToString(loc common.Location) string {
	return fmt.Sprintf("%d-%d", loc.Region(), loc.Zone())
}

// Network configurations, only one region supported for each chain
// todo 记得随着链的扩展，需要增添一些新的RPC
var networkConfigs = map[Network]NetworkConfig{
	Colosseum: {
		ChainID: big.NewInt(9000),
		RPCURLs: map[string]string{
			"0-0": "https://rpc.quai.network/cyprus1/",
		},
	},
	Garden: {
		ChainID: big.NewInt(12000),
		RPCURLs: map[string]string{
			"0-0": "https://rpc.quai.network/cyprus1/",
		},
	},
	Local: {
		ChainID: big.NewInt(1337),
		RPCURLs: map[string]string{
			"0-0": "http://localhost:9200",
		},
	},
	Orchard: {
		ChainID: big.NewInt(15000),
		RPCURLs: map[string]string{
			"0-0": "http://localhost:9200",
		},
	},
	Lighthouse: {
		ChainID: big.NewInt(17000),
		RPCURLs: map[string]string{
			"0-0": "http://localhost:9200",
		},
	},
}

// Wallet represents a wallet that can send both Quai and Qi transactions
type Wallet struct {
	privateKey *ecdsa.PrivateKey
	client     *ethclient.Client
	chainID    *big.Int
	location   common.Location
	network    Network
}

// NewWalletFromKey creates a new wallet instance from a Key
func NewWalletFromKey(key *keystore.Key, network Network) (*Wallet, error) {
	netConfig, ok := networkConfigs[network]
	if !ok {
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	// Get location from address bytes
	location := common.LocationFromAddressBytes(key.Address.Bytes())

	// Get RPC URL for the location
	rpcURL, ok := netConfig.RPCURLs[locationToString(location)]
	if !ok {
		return nil, fmt.Errorf("unsupported location %v for network %s", location, network)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %v", err)
	}

	return &Wallet{
		privateKey: key.PrivateKey,
		client:     client,
		chainID:    netConfig.ChainID,
		location:   location,
		network:    network,
	}, nil
}


// NewWalletFromPrivateKeyString creates a new wallet instance from a private key string
func NewWalletFromPrivateKeyString(privKeyHex string, network Network, location common.Location) (*Wallet, error) {
	netConfig, ok := networkConfigs[network]
	if !ok {
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	// Get RPC URL for the location
	rpcURL, ok := netConfig.RPCURLs[locationToString(location)]
	if !ok {
		return nil, fmt.Errorf("unsupported location %v for network %s", location, network)
	}

	// Convert private key string to ECDSA private key
	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node: %v", err)
	}

	return &Wallet{
		privateKey: privateKey,
		client:     client,
		chainID:    netConfig.ChainID,
		location:   location,
		network:    network,
	}, nil
}

// GetAddress returns the wallet's address
func (w *Wallet) GetAddress() common.Address {
	publicKey := w.privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA, w.location)
}

// GetChainID returns the current chain ID from the client
func (w *Wallet) GetChainID(ctx context.Context) (*big.Int, error) {
	chainID, err := w.client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}
	return chainID, nil
}

// GetBalance returns the wallet's balance
func (w *Wallet) GetBalance(ctx context.Context) (*big.Int, error) {
	address := w.GetAddress()
	return w.client.BalanceAt(ctx, address.MixedcaseAddress(), nil)
}

// SendQuai sends a Quai transaction
func (w *Wallet) SendQuai(ctx context.Context, to common.Address, amount *big.Int) (*types.Transaction, error) {
	from := w.GetAddress()
	fromMixedCase := from.MixedcaseAddress()

	// Get the nonce
	nonce, err := w.client.PendingNonceAt(ctx, fromMixedCase)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}
	fmt.Println("nonce", nonce)

	cnt, err := w.client.PendingTransactionCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}
	fmt.Println("cnt", cnt)

	// Get gas price
	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}

	tx := types.NewTx(&types.QuaiTx{
		ChainID:    w.chainID,
		Nonce:      0,
		GasPrice:   new(big.Int).Mul(gasPrice, big.NewInt(90)),
		MinerTip:   big.NewInt(1),
		Gas:        42000, // Standard transfer gas * 2
		To:         &to,
		Value:      amount,
		Data:       nil,
		AccessList: types.AccessList{},
	})

	w.printTxDetails(tx)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewSigner(w.chainID, w.location), w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	w.printTxDetails(signedTx)
	fmt.Println("tx hash", tx.Hash().Hex())
	fmt.Println("signed tx hash", signedTx.Hash().Hex())
	// Send the transaction
	err = w.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}

	// Wait for transaction receipt
	receipt, err := w.WaitForReceipt(ctx, signedTx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %v", err)
	}

	// Check if transaction was successful
	if receipt.Status == 0 {
		return nil, fmt.Errorf("transaction failed: %s", signedTx.Hash().Hex())
	}

	return signedTx, nil
}

// SendQi sends a Qi transaction
func (w *Wallet) SendQi(ctx context.Context, to common.Address, amount uint8) (*types.Transaction, error) {
	// Convert private key to btcec format for Schnorr signing
	privKeyBytes := crypto.FromECDSA(w.privateKey)
	btcecPrivKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)

	// Create TxOut
	txOut := types.NewTxOut(amount, to.Bytes(), big.NewInt(0))

	// Create QiTx
	qiTx := &types.QiTx{
		ChainID: w.chainID,
		TxOut:   types.TxOuts{*txOut},
		// Note: TxIn needs to be populated with actual UTXO data
	}
	tx := types.NewTx(qiTx)

	// Sign the transaction with Schnorr signature
	sig, err := schnorr.Sign(btcecPrivKey, tx.Hash().Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Set the signature
	qiTx.Signature = sig

	// Send the transaction
	err = w.client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}

	return tx, nil
}

// WaitForReceipt waitForReceipt waits for transaction receipt with timeout
func (w *Wallet) WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	retry := 0
	maxRetries := 30 // 等待大约5分钟 (30 * 10秒)

	for {
		receipt, err := w.client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		fmt.Println(err)

		retry++
		if retry >= maxRetries {
			return nil, fmt.Errorf("timeout waiting for transaction receipt after %d attempts", maxRetries)
		}

		// 等待10秒后重试
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			continue
		}
	}
}

// Close closes the client connection
func (w *Wallet) Close() {
	w.client.Close()
}

// printTxDetails prints transaction details with optional signature info
func (w *Wallet) printTxDetails(tx *types.Transaction) {
	// Check if transaction is signed by looking at signature values
	V, R, S := tx.GetEcdsaSignatureValues()
	isSigned := R.Sign() != 0 && S.Sign() != 0

	prefix := "Transaction"
	if isSigned {
		prefix = "Signed Transaction"
	}
	fmt.Printf("\n%s Details:\n", prefix)
	fmt.Printf("  Chain ID: %v\n", tx.ChainId())
	fmt.Printf("  Nonce: %v\n", tx.Nonce())
	fmt.Printf("  Gas Price: %v wei\n", tx.GasPrice())
	fmt.Printf("  Gas Limit: %v\n", tx.Gas())
	fmt.Printf("  To: %v\n", tx.To().Hex())
	fmt.Printf("  Value: %v wei\n", tx.Value())
	fmt.Printf("  Data: %x\n", tx.Data())
	fmt.Printf("  Hash: %v\n", tx.Hash().Hex())

	if isSigned {
		// Print signature values
		fmt.Printf("\nSignature Values:\n")
		fmt.Printf("  V: %v\n", V)
		fmt.Printf("  R: %v\n", R)
		fmt.Printf("  S: %v\n", S)

		// Get sender address from signature
		signer := types.NewSigner(w.chainID, w.location)
		if from, err := types.Sender(signer, tx); err == nil {
			fmt.Printf("  Recovered From Address: %v\n", from.Hex())
		}
	}
}
