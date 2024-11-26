package main

import (
	"fmt"

	"quai-transfer/config"
	"quai-transfer/keystore"
	"quai-transfer/utils"

	"github.com/spf13/cobra"
)

var (
	protocol string
	location string
)

var createWalletCmd = &cobra.Command{
	Use:     WalletCmdName + " [-p|--protocol quai|qi] [-l|--location zone-region]",
	Short:   WalletCmdShortDesc,
	RunE:    runCreateWallet,
	Version: Version,
}

func init() {
	flags := createWalletCmd.Flags()
	flags.StringVarP(&protocol, "protocol", "p", "quai", "Protocol type (quai/qi)")
	flags.StringVarP(&location, "location", "l", "0-0", "Location in format zone-region")
	flags.SortFlags = false
}

func runCreateWallet(cmd *cobra.Command, args []string) error {
	ks, err := keystore.NewKeyManager(keyDir)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	normalizedProtocol, err := utils.ValidateProtocol(protocol)
	if err != nil {
		return err
	}

	loc := config.StringToLocation(location)
	if err != nil {
		return fmt.Errorf("invalid location format: %w", err)
	}

	address, err := ks.CreateNewKey(loc, normalizedProtocol)
	if err != nil {
		return fmt.Errorf("failed to create new key: %w", err)
	}

	fmt.Printf("Creating new wallet with address: %s\n", address.Hex())

	return nil
}
