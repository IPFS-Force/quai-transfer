package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runRootCmd(cmd *cobra.Command, args []string) error {
	// Since we want to require subcommands, this function shouldn't be called directly
	return fmt.Errorf("a subcommand is required")
}
