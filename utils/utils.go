package utils

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"quai-transfer/types"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/shopspring/decimal"
)

func ParseTransferCSV(filepath string) ([]*wtypes.TransferEntry, error) {
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

	transfers := make([]*wtypes.TransferEntry, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) != len(expectedHeaders) {
			return nil, fmt.Errorf("invalid record length: %v", record)
		}

		minerAccountID, err := strconv.ParseUint(record[5], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse miner_account_id: %w", err)
		}

		aggregateIds := make([]int64, 0)
		for _, id := range strings.Fields(record[4]) {
			aggregateId, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse aggregate_id: %w", err)
			}
			aggregateIds = append(aggregateIds, aggregateId)
		}

		id, err := strconv.ParseInt(record[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id: %w", err)
		}

		transfer := &wtypes.TransferEntry{
			ID:             int32(id),
			MinerAccount:   record[1],
			Value:          decimal.RequireFromString(record[2]),
			ToAddress:      record[3],
			AggregateIds:   aggregateIds,
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
