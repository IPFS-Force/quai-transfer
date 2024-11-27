package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/dominant-strategies/go-quai/common"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"quai-transfer/config"
	"quai-transfer/dal/models"
	"quai-transfer/keystore"
	"quai-transfer/utils"
	"quai-transfer/wallet"
)

// TODO 7. gas price及minertip 设置为 最优值，省钱
// TODO 8. 计算出手续费的真实费用
// TODO 13. 查看SDK中转账有什么必要的字段，为什么gas price 浏览器中产生不了。且内部错误

func main() {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return
	}
	fmt.Println(cfg)
	utils.Json(cfg)
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(postgres.Open(cfg.InterDSN), config)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&models.Transaction{}); err != nil {
		log.Fatalf("failed to migrate block table: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// set keystore directory
	keystoreDir := filepath.Join(homeDir, ".quai", "keystore")

	// create KeyManager instance
	km, err := keystore.NewKeyManager(keystoreDir)
	if err != nil {
		log.Fatal(err)
	}

	// create new private key
	create_address, err := km.CreateNewKey(common.Location{0, 0}, "quai")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created new account: %s\n", create_address.Hex())

	// load private key
	key, err := km.LoadKey(create_address)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully loaded private key for address: %s\n", key.Address.Hex())

	// 1. create wallet instance
	privateKey := "ba071aefbc898130b2c83e3235a2b12d07312ca3467b2ee9a093ab4dd5af7cc2"

	w, err := wallet.NewWalletFromPrivateKeyString(
		privateKey,
		cfg,
	)
	if err != nil {
		log.Fatalf("create wallet failed: %v", err)
	}
	defer w.Close()

	// 2. get wallet address
	address := w.GetAddress()
	fmt.Printf("wallet address: %s\n", address.Hex())

	// 3. get wallet balance
	ctx := context.Background()
	balance, err := w.GetBalance(ctx)
	if err != nil {
		log.Fatalf("get balance failed: %v", err)
	}
	fmt.Printf("wallet balance: %s wei\n", balance.String())

	// 4. prepare transaction parameters
	toAddress := common.HexToAddress("0x000F82F8e14298aD129E8b0FC5dd76e10C9F02B8", w.GetLocation())
	amount := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e17)) // 1 QUAI = 10^18 wei

	// 5. send transaction
	tx, err := w.SendQuai(ctx, toAddress, amount)
	if err != nil {
		log.Fatalf("send transaction failed: %v", err)
	}

	time.Sleep(1000 * time.Second)
	// 6. print transaction hash
	fmt.Printf("transaction sent, transaction hash: %s\n", tx.Hash().Hex())
}
