package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"quai-transfer/keystore"
)

var importKeyCmd = &cobra.Command{
	Use:   ImportCmdName,
	Short: ImportCmdShortDesc,
	RunE:  runImportKey,
}

func runImportKey(cmd *cobra.Command, args []string) error {
	// Initialize keystore
	ks, err := keystore.NewKeyManager(keyDir)
	if err != nil {
		return fmt.Errorf("failed to initialize keystore: %w", err)
	}

	// Import the private key
	address, err := ks.ImportPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to import private key: %w", err)
	}

	fmt.Printf("Successfully imported key with address: %s\n", address.Hex())
	return nil
}
