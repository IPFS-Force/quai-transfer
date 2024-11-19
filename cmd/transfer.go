package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dominant-strategies/go-quai/common"
	"quai-transfer/config"
	"quai-transfer/keystore"
	"quai-transfer/utils"
	"quai-transfer/wallet"

	"github.com/spf13/cobra"
)

var (
	csvFile string
	pkFile  string
)

var transferCmd = &cobra.Command{
	Use:     TransferCmdName + " [-f|--csv /path/to/csv_file] [-p|--pk_file /path/to/private_key.json]",
	Short:   TransferCmdShortDesc,
	RunE:    runTransfer,
	Version: Version,
}

func init() {
	flags := transferCmd.Flags()
	flags.StringVarP(&csvFile, "csv", "f", "", "CSV file containing transfer details")
	flags.StringVarP(&pkFile, "pk_file", "p", "", "Private key file path")

	flags.SortFlags = false

	_ = transferCmd.MarkFlagRequired("csv")
}

func runTransfer(cmd *cobra.Command, args []string) error {
	var (
		err error
		key *keystore.Key
	)

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Initialize keystore
	ks, err := keystore.NewKeyManager(keyDir)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	if pkFile != "" {
		fmt.Printf("Loading key from private key file: %s\n", pkFile)
		key, err = ks.LoadFile(pkFile)
		if err != nil {
			return fmt.Errorf("failed to load key from private key file: %w", err)
		}
	} else {
		fmt.Printf("Loading key from config file: %s\n", cfg.KeyFile)
		key, err = ks.LoadFile(cfg.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to load key from config file: %w", err)
		}
	}
	fmt.Printf("Loaded key with address: %s\n", key.Address.Hex())

	// Create wallet instance
	w, err := wallet.NewWalletFromKey(key, cfg)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}
	defer w.Close()

	// 获取钱包余额
	ctx := context.Background()
	balance, err := w.GetBalance(ctx)
	if err != nil {
		log.Fatalf("获取余额失败: %v", err)
	}
	fmt.Printf("钱包余额: %s wei\n", balance.String())

	// Parse CSV file
	transferEntries, err := utils.ParseTransferCSV(csvFile)
	if err != nil {
		return fmt.Errorf("failed to parse CSV file: %w", err)
	}

	// Check balance
	if err := utils.CheckBalance(ctx, w, transferEntries); err != nil {
		return fmt.Errorf("insufficient balance: %w", err)
	}

	skipcnt := 0
	successcnt := 0
	failedcnt := 0

	// Process Quai transfers
	// todo: 需要处理多个类型的情况（统一用transfer来做，根据Protocol来决定 Switch case）
	for _, transfer := range transferEntries {
		if !w.IsValidQuaiAddress(transfer.ToAddress) {
			skipcnt++
			logging.Warnf("skip transfer: %s due to invalid Quai address", transfer.ToAddress)
			continue
		}
		if _, err := w.SendQuai(ctx, common.HexToAddress(transfer.ToAddress, w.GetLocation()), transfer.Value.BigInt()); err != nil {
			logging.Warnf("Transfer failed for miner account %s: %v", transfer.MinerAccount, err)
			failedcnt++
			continue
		}
		successcnt++
	}
	logging.Infof("Transfer completed, skipped: %d, success: %d, failed: %d", skipcnt, successcnt, failedcnt)

	time.Sleep(20 * time.Minute)

	return nil
}
