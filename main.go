package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"quai-transfer/config"
	"quai-transfer/dal/models"
	"quai-transfer/utils"
)

// TODO
// TODO	 1. 把sendQuai进行并发执行
// TODO 2. 把receipt 用异步的方式去等待，并把正确的结果保存在（中间回传）数据库，失败的交易也保存下来。进行计数，并把所有失败交易的接受地址和失败原因保存下来。

// TODO 4. 把调试用标记来打开或关掉

// TODO 6. 命令行工具可以多次重复执行，但是已经发送的交易不会再去发。注意不要多发了
// TODO 7. gas price及minertip 设置为 最优值，省钱
// TODO 8. 计算出手续费的真实费用

// TODO 12. 检查wallet指定的Protocol是否和转账地址的Protocol一致
// TODO 13. 查看SDK中转账有什么必要的字段，为什么gas price 浏览器中产生不了。且内部错误
// todo 14. 检查事务一致性（先插入再执行，保证不会多次转账。结果的话可以多次确认的）

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
		fmt.Errorf("failed to connect database: %v", err)
	}

	// 迁移 Block 表
	if err := db.AutoMigrate(&models.Transaction{}); err != nil {
		fmt.Errorf("failed to migrate block table: %v", err)
	}
}

//func main() {
//	// 1. 创建钱包实例
//	privateKey := "ba071aefbc898130b2c83e3235a2b12d07312ca3467b2ee9a093ab4dd5af7cc2"
//	cfg, err := config.LoadConfig("")
//	if err != nil {
//		return
//	}
//	utils.Json(cfg)
//
//	w, err := wallet.NewWalletFromPrivateKeyString(
//		privateKey,
//		cfg,
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
//	toAddress := common.HexToAddress("0x000F82F8e14298aD129E8b0FC5dd76e10C9F02B8", w.GetLocation()) // 例如: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
//	amount := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e17))                                     // 1 QUAI = 10^18 wei
//
//	// 5. 发送交易
//	tx, err := w.SendQuai(ctx, toAddress, amount)
//	if err != nil {
//		log.Fatalf("发送交易失败: %v", err)
//	}
//
//	time.Sleep(1000 * time.Second)
//	// 6. 打印交易哈希
//	fmt.Printf("交易已发送，交易哈希: %s\n", tx.Hash().Hex())
//}

//func main() {
//
//	homeDir, err := os.UserHomeDir()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 设置keystore目录
//	keystoreDir := filepath.Join(homeDir, ".quai", "keystore")
//
//	// 创建KeyManager实例
//	km, err := keystore.NewKeyManager(keystoreDir)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 创建新私钥 (假设我们要创建ID范围在0-255之间的私钥)
//	address, err := km.CreateNewKey(common.Location{0, 0})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Created new account: %s\n", address.Hex())
//
//	// 加载私钥
//	key, err := km.LoadKey(address)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Successfully loaded private key for address: %s\n", key.Address.Hex())
//}
