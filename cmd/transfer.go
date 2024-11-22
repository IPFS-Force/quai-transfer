package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"quai-transfer/config"
	"quai-transfer/keystore"
	wtypes "quai-transfer/types"
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

	// get wallet balance
	ctx := context.Background()
	balance, err := w.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %v", err)
	}
	fmt.Printf("Wallet balance: %s Quai\n", utils.ToQuai(balance.String()))

	// Parse CSV file
	transferEntries, err := utils.ParseTransferCSV(csvFile)
	if err != nil {
		return fmt.Errorf("failed to parse CSV file: %w", err)
	}

	// Check if address have enough balance for all entries
	if err := wallet.CheckBalance(ctx, w, transferEntries); err != nil {
		return fmt.Errorf("insufficient balance: %w", err)
	}

	invalidCnt := 0
	successCnt := 0
	failedCnt := 0
	processedCnt := 0

	// Process Quai transfers
	// todo: 需要处理多个类型的情况（统一用transfer来做，根据Protocol来决定 Switch case）
	for _, entry := range transferEntries {
		if !w.IsValidQuaiAddress(entry.ToAddress) {
			invalidCnt++
			log.Fatalf("skip transfer: <%s> due to invalid Quai address", entry.ToAddress)
			continue
		}
		if err := w.ProcessEntry(ctx, entry); err != nil {
			if errors.Is(err, wtypes.ErrAlreadyProcessed) {
				log.Printf("\n⏭️ TRANSFER SKIPPED ⏭️\nMiner Account: %s\nEntry ID: %d\nReason: Already processed\n", entry.MinerAccount, entry.ID)
				processedCnt++
				continue
			}
			log.Printf("\n❌ TRANSFER FAILED ❌\nMiner Account: %s\nEntry ID: %d\nError: %v\n", entry.MinerAccount, entry.ID, err)
			failedCnt++
			continue
		}
		log.Printf("\n✅ TRANSFER SUCCESSFUL ✅\nMiner Account: %s\nEntry ID: %d\nTransferred: %s Quai\n", entry.MinerAccount, entry.ID, utils.ToQuai(entry.Value.String()))
		successCnt++
	}
	log.Printf("Transfer completed, total: %d, success: %d, failed: %d, processed: %d, invalid address: %d", len(transferEntries), successCnt, failedCnt, processedCnt, invalidCnt)
	return nil
}
