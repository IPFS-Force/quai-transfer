package main

import (
	"context"
	"fmt"

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
	utils.Json(cfg)

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

	ctx := context.Background()
	balance, err := w.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %v", err)
	}
	fmt.Printf("Wallet balance: %s Quai\n", utils.ToQuai(balance.String()))

	transferEntries, err := utils.ParseTransferCSV(csvFile)
	if err != nil {
		return fmt.Errorf("failed to parse CSV file: %w", err)
	}

	// Check if address have enough balance for all entries
	if err := wallet.CheckBalance(ctx, w, transferEntries); err != nil {
		return fmt.Errorf("insufficient balance: %w", err)
	}

	// todo: 需要处理多个类型的情况（统一用transfer来做，根据Protocol来决定 Switch case）
	w.ProcessBatchEntry(ctx, transferEntries)
	return nil
}
