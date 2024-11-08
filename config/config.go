package config

import (
	"io"
	"os"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/pelletier/go-toml/v2"
)

var Conf = &Config{}

type Config struct {
	InterDSN  string          `toml:"dsn"`
	WalletDSN string          `toml:"wallet_dsn"` // 钱包的dsn
	Network   string          `toml:"network"`    // 网络名称（MainNet：主网，TestNet：测试网）
	Rpc       string          `toml:"rpc"`        // rpc配置
	Protocol  string          `toml:"protocol"`   // 协议名称
	Location  common.Location `toml:"location"`   // 所在链位置
}

func Init(file string) (err error) {
	f, err := os.Open(file)
	if err != nil {
		return
	}

	d, err := io.ReadAll(f)
	if err != nil {
		return
	}

	err = toml.Unmarshal(d, Conf)
	if err != nil {
		return
	}

	return
}
