package main

import (
	"log"
)

var (
	// Configuration file path
	configFile string

	// Version information (set via ldflags)
	Version string

	// Key directory path
	keyDir string = "./.keystore"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

const (
	// AppName Application names and descriptions
	AppName   = "quai-wallet"
	ShortDesc = "Quai blockchain wallet management tool"

	// WalletCmdName Wallet command constants
	WalletCmdName      = "create"
	WalletCmdShortDesc = "Create and manage Quai blockchain wallets"

	// TransferCmdName Transfer command constants
	TransferCmdName      = "transfer"
	TransferCmdShortDesc = "Transfer Quai or Qi tokens in batches"

	// ImportCmdName Import command constants
	ImportCmdName      = "import"
	ImportCmdShortDesc = "Import a private key into the keystore"
)
