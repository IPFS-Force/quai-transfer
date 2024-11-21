package wallet

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strings"
	"time"

	"quai-transfer/config"
	"quai-transfer/dal"
	"quai-transfer/dal/models"
	"quai-transfer/keystore"
	wtypes "quai-transfer/types"

	"github.com/dominant-strategies/go-quai/common/hexutil"
	"google.golang.org/protobuf/proto"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/dominant-strategies/go-quai/crypto"
	"github.com/dominant-strategies/go-quai/quaiclient/ethclient"
	"github.com/shopspring/decimal"
)

// Ensure Wallet implements WalletFunc interface
var _ WalletFunc = (*Wallet)(nil)

const (
	GasLimit = 420000
	MinerTip = 1000
)

// ChainIDMapping holds the expected and actual chain IDs
type ChainIDMapping struct {
	Expected *big.Int
	Actual   *big.Int
}

// Wallet represents a wallet that can send both Quai and Qi transactions
type Wallet struct {
	privateKey *ecdsa.PrivateKey
	client     *ethclient.Client
	chainID    *ChainIDMapping
	location   common.Location
	network    wtypes.Network
	address    common.Address
	txDAL      *dal.TransactionDAL
	config     *config.Config
}

func (w *Wallet) GetLocation() common.Location {
	return w.location
}

func (w *Wallet) GetBalance(ctx context.Context) (*big.Int, error) {
	address := w.GetAddress()
	return w.client.BalanceAt(ctx, address.MixedcaseAddress(), nil)
}

func (w *Wallet) BroadcastTransaction(ctx context.Context, tx *types.Transaction) error {
	protoTx, err := tx.ProtoEncode()
	if err != nil {
		return err
	}
	data, err := proto.Marshal(protoTx)
	if err != nil {
		return err
	}
	log.Println("transaction raw data:", hexutil.Encode(data))
	return w.client.SendTransaction(ctx, tx)
}

func (w *Wallet) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return w.client.SuggestGasPrice(ctx)
}

func (w *Wallet) GetNonce(ctx context.Context) (uint64, error) {
	return w.client.PendingNonceAt(ctx, w.GetAddress().MixedcaseAddress())
}

func (w *Wallet) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return w.client.TransactionReceipt(ctx, txHash)
}

func (w *Wallet) Close() {
	w.client.Close()
}

func (w *Wallet) GetAddress() common.Address {
	return w.address
}

// GetChainID returns the current chain ID from the client
func (w *Wallet) GetChainID(ctx context.Context) (*big.Int, error) {
	if w.chainID.Actual == nil {
		if err := w.verifyChainID(ctx); err != nil {
			return nil, err
		}
	}
	return w.chainID.Actual, nil
}

// initClient initializes the wallet's client connection
func (w *Wallet) initClient(network wtypes.Network) error {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	netConfig, ok := cfg.Networks[network]
	if !ok {
		return fmt.Errorf("unsupported network: %s", network)
	}

	// Get location from wallet's address
	location := w.calculateLocation()

	// Get RPC URL for the location
	rpcURL, ok := netConfig.RPCURLs[locationToString(location)]
	if !ok {
		return fmt.Errorf("unsupported location %v for network %s", location, network)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to node: %v", err)
	}

	*w = Wallet{
		client:     client,
		chainID:    &ChainIDMapping{Expected: netConfig.ChainID},
		location:   location,
		network:    network,
		config:     cfg,
		privateKey: w.privateKey,
		address:    w.address,
		txDAL:      w.txDAL,
	}

	return nil
}

// calculateLocation calculates the location from the wallet's address
func (w *Wallet) calculateLocation() common.Location {
	return common.LocationFromAddressBytes(w.address.Bytes())
}

// NewWalletFromKey creates a new wallet instance from a Key
func NewWalletFromKey(key *keystore.Key, cfg *config.Config) (*Wallet, error) {
	dal.DBInit(cfg)

	wallet := &Wallet{
		privateKey: key.PrivateKey,
		txDAL:      dal.NewTransactionDAL(dal.InterDB),
		address:    key.Address,
	}

	// Initialize client and other fields
	if err := wallet.initClient(cfg.Network); err != nil {
		return nil, err
	}

	if err := wallet.verifyChainID(context.Background()); err != nil {
		wallet.Close()
		return nil, err
	}

	return wallet, nil
}

// NewWalletFromPrivateKeyString creates a new wallet instance from a private key string
func NewWalletFromPrivateKeyString(privKeyHex string, cfg *config.Config) (*Wallet, error) {
	dal.DBInit(cfg)

	privateKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	wallet := &Wallet{
		privateKey: privateKey,
		txDAL:      dal.NewTransactionDAL(dal.InterDB),
	}

	// Calculate the address first
	wallet.address = wallet.calculateAddress()

	// Initialize client and other fields
	if err := wallet.initClient(cfg.Network); err != nil {
		return nil, err
	}

	if err := wallet.verifyChainID(context.Background()); err != nil {
		wallet.Close()
		return nil, err
	}

	return wallet, nil
}

// SendQuai sends a Quai transaction asynchronously
func (w *Wallet) SendQuai(ctx context.Context, to common.Address, amount *big.Int) (*types.Transaction, error) {
	from := w.GetAddress()

	nonce, err := w.GetNonce(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}
	fmt.Printf("Nonce: %d\n", nonce)

	gasPrice, err := w.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}
	fmt.Printf("Gas price: %v\n", gasPrice)

	tx := types.NewTx(&types.QuaiTx{
		ChainID:    w.chainID.Actual,
		Nonce:      nonce,
		GasPrice:   gasPrice,
		MinerTip:   big.NewInt(MinerTip),
		Gas:        GasLimit,
		To:         &to,
		Value:      amount,
		Data:       nil,
		AccessList: types.AccessList{},
	})
	w.printTxDetails(tx)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewSigner(w.chainID.Actual, w.location), w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	w.printTxDetails(signedTx)

	txRecord := &models.Transaction{
		Payer:     from.Hex(),
		ToAddress: to.Hex(),
		TxHash:    signedTx.Hash().Hex(),
		Nonce:     nonce,
		Value:     decimal.NewFromBigInt(amount, 0),
		GasLimit:  decimal.NewFromInt(int64(signedTx.Gas())),
		GasPrice:  decimal.NewFromBigInt(signedTx.GasPrice(), 0),
		Status:    models.Generated, // pending
		CreatedAt: time.Now(),
	}

	if err = w.txDAL.CreateTransaction(ctx, txRecord); err != nil {
		return nil, fmt.Errorf("failed to create transaction record: %v", err)
	}
	fmt.Printf("Created transaction record: %d\n", txRecord.ID)

	if err := w.BroadcastTransaction(ctx, signedTx); err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}
	fmt.Printf("transaction: %s has been broadcasted\n", signedTx.Hash().Hex())

	// Start receipt monitoring
	if err := w.MonitorAndConfirmTransaction(context.Background(), signedTx); err != nil {
		return nil, err
	}

	return signedTx, nil
}

// MonitorAndConfirmTransaction monitors the transaction and updates the database when confirmed
func (w *Wallet) MonitorAndConfirmTransaction(ctx context.Context, tx *types.Transaction) (err error) {
	receipt, err := w.WaitForReceipt(ctx, tx.Hash())
	if err != nil {
		// Log error but don't return since this is async
		fmt.Printf("Error waiting for receipt: %v\n", err)
		return err
	}

	// Print receipt details for logging
	w.printReceiptDetails(receipt)

	gasUsedAmount := decimal.NewFromInt(int64(receipt.GasUsed)).Mul(decimal.NewFromBigInt(tx.GasPrice(), 0))

	// Update transaction record with confirmation details
	err = w.txDAL.UpdateTransactionStatus(
		ctx,
		tx.Hash().Hex(),
		gasUsedAmount,
		receipt,
	)
	if err != nil {
		fmt.Printf("Error updating transaction status: %v\n", err)
		return err
	}
	return nil
}

func (w *Wallet) CheckTransactionAndConfirm(ctx context.Context, tx *types.Transaction) (err error) {
	receipt, err := w.GetTransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return err
	}

	// Print receipt details for logging
	w.printReceiptDetails(receipt)

	gasUsedAmount := decimal.NewFromInt(int64(receipt.GasUsed)).Mul(decimal.NewFromBigInt(tx.GasPrice(), 0))

	// Update transaction record with confirmation details
	err = w.txDAL.UpdateTransactionStatus(
		ctx,
		tx.Hash().Hex(),
		gasUsedAmount,
		receipt,
	)
	if err != nil {
		fmt.Printf("Error updating transaction status: %v\n", err)
		return err
	}
	fmt.Printf("Check transaction %s has been confirmed in database\n", tx.Hash().Hex())
	return nil
}

// SendQi sends a Qi transaction
func (w *Wallet) SendQi(ctx context.Context, to common.Address, amount uint8) (*types.Transaction, error) {
	// Convert private key to btcec format for Schnorr signing
	privKeyBytes := crypto.FromECDSA(w.privateKey)
	btcecPrivKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)

	txOut := types.NewTxOut(amount, to.Bytes(), big.NewInt(0))

	qiTx := &types.QiTx{
		ChainID: w.chainID.Actual,
		TxOut:   types.TxOuts{*txOut},
		// Note: TxIn needs to be populated with actual UTXO data
	}
	tx := types.NewTx(qiTx)

	// Sign the transaction with Schnorr signature
	sig, err := schnorr.Sign(btcecPrivKey, tx.Hash().Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	qiTx.Signature = sig

	err = w.BroadcastTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %v", err)
	}
	return tx, nil
}

// WaitForReceipt waits for transaction receipt with timeout
func (w *Wallet) WaitForReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	retry := 0
	maxRetries := 30 // Wait for about 5 minutes (30 * 10 seconds)

	for {
		receipt, err := w.GetTransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		retry++
		if retry >= maxRetries {
			return nil, fmt.Errorf("timeout waiting for transaction receipt after %d attempts", maxRetries)
		}

		// Wait 10 seconds before retrying
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
			continue
		}
	}
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
		signer := types.NewSigner(w.chainID.Actual, w.location)
		if from, err := types.Sender(signer, tx); err == nil {
			fmt.Printf("  Recovered From Address: %v\n", from.Hex())
		}
	}
}

// printReceiptDetails prints transaction receipt details
func (w *Wallet) printReceiptDetails(receipt *types.Receipt) {
	fmt.Printf("\nTransaction Receipt Details:\n")
	fmt.Printf("  Type: %v\n", receipt.Type)
	if len(receipt.PostState) > 0 {
		fmt.Printf("  Post State: %x\n", receipt.PostState)
	}
	fmt.Printf("  Status: %v (%s)\n", receipt.Status, getStatusString(receipt.Status))
	fmt.Printf("  Transaction Hash: %v\n", receipt.TxHash.Hex())
	fmt.Printf("  Block Hash: %v\n", receipt.BlockHash.Hex())
	fmt.Printf("  Block Number: %v\n", receipt.BlockNumber)
	fmt.Printf("  Transaction Index: %v\n", receipt.TransactionIndex)
	fmt.Printf("  Gas Used: %v\n", receipt.GasUsed)
	fmt.Printf("  Cumulative Gas Used: %v\n", receipt.CumulativeGasUsed)

	if receipt.ContractAddress != (common.Address{}) {
		fmt.Printf("  Contract Address: %v\n", receipt.ContractAddress.Hex())
	}

	if len(receipt.Logs) > 0 {
		fmt.Printf("\n  Event Logs (%d):\n", len(receipt.Logs))
		for i, log := range receipt.Logs {
			fmt.Printf("    Log #%d:\n", i)
			fmt.Printf("      Address: %v\n", log.Address.Hex())
			fmt.Printf("      Topics:\n")
			for j, topic := range log.Topics {
				fmt.Printf("        [%d]: %v\n", j, topic.Hex())
			}
			fmt.Printf("      Data: %x\n", log.Data)
		}
	}

	if len(receipt.OutboundEtxs) > 0 {
		fmt.Printf("\n  Outbound External Transactions (%d):\n", len(receipt.OutboundEtxs))
		for i, etx := range receipt.OutboundEtxs {
			fmt.Printf("    ETX #%d:\n", i)
			fmt.Printf("      Hash: %v\n", etx.Hash().Hex())
			if etx.To() != nil {
				fmt.Printf("      To: %v\n", etx.To().Hex())
			}
			fmt.Printf("      Value: %v\n", etx.Value())
		}
	}
}

// getStatusString converts receipt status to a human-readable string
func getStatusString(status uint64) string {
	switch status {
	case types.ReceiptStatusSuccessful:
		return "Success"
	case types.ReceiptStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// verifyChainID verifies if the chain ID is correct with the expected chain ID
func (w *Wallet) verifyChainID(ctx context.Context) error {
	actualChainID, err := w.client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID from client: %v", err)
	}

	w.chainID.Actual = actualChainID

	if w.chainID.Expected.Cmp(actualChainID) != 0 {
		return fmt.Errorf("chain ID mismatch: expected %v, got %v", w.chainID.Expected, actualChainID)
	}
	return nil
}

// calculateAddress calculates the address
func (w *Wallet) calculateAddress() common.Address {
	publicKey := w.privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA, w.location)
}

// locationToString converts a Location to a string key
func locationToString(loc common.Location) string {
	return fmt.Sprintf("%d-%d", loc.Region(), loc.Zone())
}

// IsValidAddress validate address is valid and in current chain scope
func (w *Wallet) IsValidAddress(address string) bool {
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	if !re.MatchString(address) {
		return false
	}
	addressBytes := common.FromHex(address)
	return common.IsInChainScope(addressBytes, w.location)
}

// IsValidQuaiAddress validate address is valid and in Quai ledger scope
func (w *Wallet) IsValidQuaiAddress(address string) bool {
	return w.IsValidAddress(address) && IsInQuaiLedgerScope(address)
}

// IsValidQiAddress validate address is valid and in Qi ledger scope
func (w *Wallet) IsValidQiAddress(address string) bool {
	return w.IsValidAddress(address) && IsInQiLedgerScope(address)
}

// ProcessEntry handles a single transfer entry
func (w *Wallet) ProcessEntry(ctx context.Context, entry *wtypes.TransferEntry) error {
	if !w.IsValidQuaiAddress(entry.ToAddress) {
		return fmt.Errorf("invalid Quai address: %s", entry.ToAddress)
	}

	signedTx, storedEntry, status, err := w.GetTransactionByID(ctx, entry.ID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if transaction is already confirmed
	if status == models.Confirmed {
		return wtypes.ErrAlreadyProcessed
	}

	if storedEntry != nil && !CompareEntries(entry, storedEntry) {
		return fmt.Errorf("entry mismatch for ID %d: stored entry differs from provided entry", entry.ID)
	}

	if signedTx == nil {
		fmt.Printf("entry %d: transaction not found in database, creating new transaction\n", entry.ID)
		// Create and store transaction
		signedTx, err = w.CreateTransaction(ctx, entry)
		if err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
	}

	w.printTxDetails(signedTx)
	txHash := signedTx.Hash().Hex()

	err = w.BroadcastTransaction(ctx, signedTx)
	if err == nil {
		fmt.Printf("transaction: %s has been broadcasted\n", txHash)
		return w.MonitorAndConfirmTransaction(ctx, signedTx)
	}

	// Handle specific error cases
	switch {
	case strings.Contains(err.Error(), "nonce too low"):
		if err := w.CheckTransactionAndConfirm(ctx, signedTx); err != nil {
			return fmt.Errorf("failed to check and confirm transaction: %w", err)
		}
		return nil

	case strings.Contains(err.Error(), "already known"):
		log.Printf("transaction: %s already known, skipping", txHash)
		return w.MonitorAndConfirmTransaction(ctx, signedTx)

	default:
		return fmt.Errorf("failed to send transaction: %w", err)
	}
}

// CreateTransaction creates a new transaction and stores it in the database
func (w *Wallet) CreateTransaction(ctx context.Context, entry *wtypes.TransferEntry) (*types.Transaction, error) {
	from := w.GetAddress()
	to := common.HexToAddress(entry.ToAddress, w.GetLocation())

	nonce, err := w.GetNonce(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}
	fmt.Printf("Nonce: %d\n", nonce)

	gasPrice, err := w.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}
	fmt.Printf("Gas price: %v\n", gasPrice)

	tx := types.NewTx(&types.QuaiTx{
		ChainID:    w.chainID.Actual,
		Nonce:      nonce,
		GasPrice:   gasPrice,
		MinerTip:   big.NewInt(MinerTip),
		Gas:        GasLimit,
		To:         &to,
		Value:      entry.Value.BigInt(),
		Data:       nil,
		AccessList: types.AccessList{},
	})

	signedTx, err := types.SignTx(tx, types.NewSigner(w.chainID.Actual, w.location), w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	txJSON, err := json.Marshal(signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %v", err)
	}

	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize entry: %v", err)
	}

	txRecord := &models.Transaction{
		ID:           entry.ID,
		MinerAccount: uint(entry.MinerAccountID),
		Payer:        from.Hex(),
		ToAddress:    to.Hex(),
		TxHash:       signedTx.Hash().Hex(),
		Nonce:        nonce,
		Value:        entry.Value,
		GasLimit:     decimal.NewFromInt(int64(signedTx.Gas())),
		GasPrice:     decimal.NewFromBigInt(signedTx.GasPrice(), 0),
		AggregateIds: entry.AggregateIds,
		Status:       models.Generated,
		CreatedAt:    time.Now(),
		Tx:           string(txJSON),
		Entry:        string(entryJSON),
	}

	if err = w.txDAL.CreateTransaction(ctx, txRecord); err != nil {
		return nil, fmt.Errorf("failed to create transaction record: %v", err)
	}

	return signedTx, nil
}

func CheckBalance(ctx context.Context, w *Wallet, transferEntries []*wtypes.TransferEntry) error {
	balance, err := w.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}
	balanceDecimal := decimal.NewFromBigInt(balance, 0)

	totalAmount := decimal.Zero
	for _, entry := range transferEntries {
		totalAmount = totalAmount.Add(entry.Value)
	}

	gasPrice, err := w.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// to make sure we have enough balance, we multiply the gas price by 10
	gasPriceDecimal := decimal.NewFromBigInt(gasPrice, 0).Mul(decimal.NewFromInt(10))

	// Calculate total gas cost for all entries ---- Standard transfer gas limit* estimate gas price * 10 * number of transfers
	estimatedGas := gasPriceDecimal.Mul(decimal.NewFromInt(GasLimit * int64(len(transferEntries))))
	totalRequired := totalAmount.Add(estimatedGas)

	if balanceDecimal.LessThan(totalRequired) {
		return fmt.Errorf("insufficient balance for transfers: have %s, need %s",
			balanceDecimal.String(), totalRequired.String())
	}
	return nil
}

// GetTransactionByID retrieves transaction details by ID
func (w *Wallet) GetTransactionByID(ctx context.Context, id int32) (*types.Transaction, *wtypes.TransferEntry, models.TxStatus, error) {
	txRecord, err := w.txDAL.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get transaction: %v", err)
	}
	if txRecord == nil {
		return nil, nil, 0, nil // Return nil if no record found
	}
	fmt.Println(txRecord)

	var tx types.Transaction
	if err := json.Unmarshal([]byte(txRecord.Tx), &tx); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to deserialize transaction: %v", err)
	}

	var entry wtypes.TransferEntry
	if err := json.Unmarshal([]byte(txRecord.Entry), &entry); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to deserialize entry: %v", err)
	}

	return &tx, &entry, txRecord.Status, nil
}

// CompareEntries compares two TransferEntry objects and returns true if they are equal
func CompareEntries(a, b *wtypes.TransferEntry) bool {
	if a == nil || b == nil {
		return a == b // Both should be nil to be equal
	}

	return a.ID == b.ID &&
		a.MinerAccountID == b.MinerAccountID &&
		a.ToAddress == b.ToAddress &&
		a.Value.Equal(b.Value)
}
