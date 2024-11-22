package main

import (
	"fmt"

	"quai-transfer/config"
	"quai-transfer/keystore"
	"quai-transfer/utils"

	"github.com/spf13/cobra"
)

var (
	num      int64
	iscsv    bool
	protocol string
)

var createWalletCmd = &cobra.Command{
	Use:     WalletCmdName + " [-p|--protocol quai|qi]",
	Short:   WalletCmdShortDesc,
	RunE:    runCreateWallet,
	Version: Version,
}

func init() {
	flags := createWalletCmd.Flags()
	flags.StringVarP(&protocol, "protocol", "p", "quai", "Protocol type (quai/qi)")
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

	normalizedProtocol, err := utils.ValidateProtocol(protocol)
	if err != nil {
		return err
	}

	address, err := ks.CreateNewKey(cfg.Location, normalizedProtocol)
	if err != nil {
		return fmt.Errorf("failed to create new key: %w", err)
	}

	fmt.Printf("Creating new wallet with address: %s\n", address.Hex())

	return nil
}
