package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:     os.Args[0] + " <subcommand> [-c|--config /path/to/config.toml]",
	Short:   ShortDesc,
	Version: Version,
	RunE:    runRootCmd,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	rootCmd.Flags().SortFlags = false
	_ = rootCmd.MarkFlagRequired("config")

	// Add subcommands
	rootCmd.AddCommand(createWalletCmd)
	rootCmd.AddCommand(transferCmd)
	rootCmd.AddCommand(importKeyCmd)

	// Require a subcommand
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
