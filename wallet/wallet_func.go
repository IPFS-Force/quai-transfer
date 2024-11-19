package wallet

// Check if needed and in what scenarios?
//type WalletFunc interface {
//	GetNowBlockNum() (uint64, error)
//	GetTransaction(num uint64) ([]types.Transaction, uint64, error)
//	GetTransactionReceipt(*types.Transaction) error
//	GetBalance(address string) (*big.Int, error)
//	GetUSDTBalanceByAPI(address string) (*big.Int, error)
//	CreateWallet() (*types.Wallet, error)
//	Transfer(privateKeyStr string, toAddress string, value *big.Int, nonce uint64) (string, string, uint64, error)
//}
//
//type QuaiWallet struct {
//	Worker   WalletFunc
//	config   config.Config
//	Protocol string
//	localDB  dal.SqliteDB // Can encapsulate all interfaces and only expose one interface
//}
