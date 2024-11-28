package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	// Configuration file path
	configFile string

	// Version information (set via ldflags)
	Version string

	// Key directory path
	keyDir string = "./.keystore"

	// Logger settings
	logFile *os.File
)

// initLogger initializes the logging system to output to both file and terminal
func initLogger() error {
	logsDir := "./logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02_15:04:05")
	logPath := filepath.Join(logsDir, fmt.Sprintf("quai-wallet-%s.log", timestamp))

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	logFile = file

	// Create multi-writer for both terminal and file output
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	flags := log.LstdFlags | log.Lshortfile

	// Replace the standard logger with our multi-output logger
	log.SetOutput(multiWriter)
	log.SetFlags(flags)

	return nil
}

// closeLogger ensures the log file is properly closed
func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

func init() {
	if err := initLogger(); err != nil {
		// If initialize error, continue with console-only logging
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		return
	}
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
