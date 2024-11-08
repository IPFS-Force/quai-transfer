package main

import (
	"fmt"
	"os"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"quai-transfer/config"
	"quai-transfer/dal"
)

var (
	log = logging.Logger("payment")
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "quai-payment"

	// ShortName is the name of the ShortName software.
	ShortName string = "payment"

	// Version is the version of the compiled software.
	Version string

	// CreateWalletName is the name of the compiled software.
	CreateWalletName string = "wallet-create"

	// CreateWalletShortName is the name of the ShortName software.
	CreateWalletShortName string = "create"
)

var (
	// Father Command
	rootCmd = &cobra.Command{
		Use:   os.Args[0] + " [-c|--config /path/to/config.toml]",
		Short: ShortName,
		Run:   runCmd,
		//Args:    cobra.ExactArgs(0),
		Version: Version,
	}
	configFile string

	// Create Wallet Command
	createWalletCmd = &cobra.Command{
		Use:   CreateWalletShortName + " [-n|--num 100] [-v|--csv true]",
		Short: CreateWalletName,
		Run:   runCreateWalletCmd,
		//Args:    cobra.ExactArgs(0),
		Version: Version,
	}
	num      int64
	iscsv    bool
	protocol string // Quai 还是 Qi
)

func init() {
	logging.SetDebugLogging()
	createWalletCmd.Flags().BoolVarP(&iscsv, "csv", "v", true, "导出csv")
	createWalletCmd.Flags().Int64VarP(&num, "num", "n", 1, "生成钱包地址个数")
	createWalletCmd.Flags().SortFlags = false
	_ = createWalletCmd.MarkFlagRequired("protocol")
	//logging.SetDebugLogging()

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径")
	rootCmd.Flags().SortFlags = false
	_ = rootCmd.MarkFlagRequired("config")
	rootCmd.AddCommand(createWalletCmd)
	//rootCmd.AddCommand(listCmd)
}

func runCmd(_ *cobra.Command, _ []string) {
	err := config.Init(configFile)

	if err != nil {
		log.Fatal(err)
	}

	dal.DBInit(config.Conf)

	db, err := dal.InterDB.DB() //这里改成生成的本地的sqlite数据库
	if err != nil {
		log.Warn("get db err: ", err)
	}

	err = db.Close()
	if err != nil {
		log.Warn("db close err:", err)
	}

}

func runCreateWalletCmd(_ *cobra.Command, _ []string) {
	err := config.Init(configFile)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
