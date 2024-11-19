package main

import (
	"fmt"

	"quai-transfer/config"
	"quai-transfer/keystore"

	"github.com/spf13/cobra"
)

var (
	num      int64
	iscsv    bool
	protocol string
)

var createWalletCmd = &cobra.Command{
	Use:     WalletCmdName + "",
	Short:   WalletCmdShortDesc,
	RunE:    runCreateWallet,
	Version: Version,
}

func init() {
	flags := createWalletCmd.Flags()
	flags.StringVarP(&protocol, "protocol", "p", "Quai", "Protocol type (Quai/Qi)")
	flags.SortFlags = false

	// _ = createWalletCmd.MarkFlagRequired("protocol")
}

func runCreateWallet(cmd *cobra.Command, args []string) error {
	var (
		err error
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

	address, err := ks.CreateNewKey(cfg.Location)
	if err != nil {
		return fmt.Errorf("failed to create new key: %w", err)
	}

	// TODO: Implement wallet creation logic
	fmt.Printf("Creating new wallet with address: %s\n", address.Hex())

	return nil
}
