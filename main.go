package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dominant-strategies/go-quai/common"
	"quai-transfer/keystore"
)

//func main() {
//	// 1. 创建钱包实例
//	privateKey := "d2b46383d05c229523b0ec58dfae49405c12ab6313c58991278e411bca7997e0"
//	location := common.Location{0, 0} // 设置位置，例如：Region 0, Zone 0
//
//	w, err := wallet.NewWalletFromPrivateKeyString(
//		privateKey,
//		wallet.Garden, // 或者使用其他网络：wallet.Garden, wallet.Local
//		location,
//	)
//	if err != nil {
//		log.Fatalf("创建钱包失败: %v", err)
//	}
//	defer w.Close()
//
//	// 2. 获取钱包地址
//	address := w.GetAddress()
//	fmt.Printf("钱包地址: %s\n", address.Hex())
//
//	// 3. 获取钱包余额
//	ctx := context.Background()
//	balance, err := w.GetBalance(ctx)
//	if err != nil {
//		log.Fatalf("获取余额失败: %v", err)
//	}
//	fmt.Printf("钱包余额: %s wei\n", balance.String())
//
//	// 4. 准备交易参数
//	toAddress := common.HexToAddress("0x000F82F8e14298aD129E8b0FC5dd76e10C9F02B8", location) // 例如: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
//	amount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18))                           // 1 QUAI = 10^18 wei
//
//	// 5. 发送交易
//	tx, err := w.SendQuai(ctx, toAddress, amount)
//	if err != nil {
//		log.Fatalf("发送交易失败: %v", err)
//	}
//
//	// 6. 打印交易哈希
//	fmt.Printf("交易已发送，交易哈希: %s\n", tx.Hash().Hex())
//}

func main() {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// 设置keystore目录
	keystoreDir := filepath.Join(homeDir, ".quai", "keystore")

	// 创建KeyManager实例
	km, err := keystore.NewKeyManager(keystoreDir)
	if err != nil {
		log.Fatal(err)
	}

	// 创建新私钥 (假设我们要创建ID范围在0-255之间的私钥)
	address, err := km.CreateNewKey(common.Location{0, 0})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created new account: %s\n", address.Hex())

	// 加载私钥
	key, err := km.LoadKey(address)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully loaded private key for address: %s\n", key.Address.Hex())
}
