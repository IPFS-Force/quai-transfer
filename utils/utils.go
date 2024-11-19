package utils

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"quai-transfer/wallet"

	"github.com/fatih/color"
	"github.com/shopspring/decimal"
)

type TransferEntry struct {
	ID             string
	MinerAccount   string
	Value          decimal.Decimal
	ToAddress      string
	AggregateIds   []string
	MinerAccountID uint64
}

func ParseTransferCSV(filepath string) ([]*TransferEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must contain at least a header row and one data row")
	}

	// Validate header
	header := records[0]
	expectedHeaders := []string{"id", "miner_account", "value", "to_address", "aggregate_ids", "miner_account_id"}
	if !validateHeaders(header, expectedHeaders) {
		return nil, fmt.Errorf("invalid CSV headers, expected: %v", expectedHeaders)
	}

	transfers := make([]*TransferEntry, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) != len(expectedHeaders) {
			return nil, fmt.Errorf("invalid record length: %v", record)
		}

		minerAccountID, err := strconv.ParseUint(record[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse miner_account_id: %w", err)
		}
		transfer := &TransferEntry{
			ID:             record[0],
			MinerAccount:   record[1],
			Value:          decimal.RequireFromString(record[2]),
			ToAddress:      record[3],
			AggregateIds:   strings.Fields(record[4]), // Split by whitespace
			MinerAccountID: minerAccountID,
		}
		transfers = append(transfers, transfer)
	}

	return transfers, nil
}

func validateHeaders(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, header := range actual {
		if strings.ToLower(header) != expected[i] {
			return false
		}
	}
	return true
}

func CheckBalance(ctx context.Context, w *wallet.Wallet, transfers []*TransferEntry) error {
	// Get wallet balance
	balance, err := w.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}
	balanceDecimal := decimal.NewFromBigInt(balance, 0)

	// Calculate total amount needed
	totalAmount := decimal.Zero
	for _, transfer := range transfers {
		totalAmount = totalAmount.Add(transfer.Value)
	}

	// Estimate gas cost
	gasPrice, err := w.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}
	gasPriceDecimal := decimal.NewFromBigInt(gasPrice, 0)

	// Multiply gas price by 10
	gasPriceDecimal = gasPriceDecimal.Mul(decimal.NewFromInt(10))

	// Calculate total gas cost for all transfers
	estimatedGas := gasPriceDecimal.Mul(decimal.NewFromInt(42000 * int64(len(transfers)))) // Standard transfer gas usage * 2 * estimate gas price * 10 * number of transfers
	totalRequired := totalAmount.Add(estimatedGas)

	if balanceDecimal.LessThan(totalRequired) {
		return fmt.Errorf("insufficient balance for transfers: have %s, need %s",
			balanceDecimal.String(), totalRequired.String())
	}
	return nil
}

func Json(a ...any) {
	color.Yellow("%s spew json: \n", runFuncPos())
	for _, v := range a {
		d, _ := json.MarshalIndent(v, "", "\t")
		fmt.Println(string(d))
	}
}

func runFuncPos() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s:%d", file, line)
}
