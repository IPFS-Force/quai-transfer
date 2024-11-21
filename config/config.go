package config

import (
	"fmt"
	"math/big"
	"strings"

	wtypes "quai-transfer/types"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/spf13/viper"
)

var GlobalLocation common.Location

// NetworkConfig holds network specific configuration
type NetworkConfig struct {
	ChainID *big.Int          `mapstructure:"chain_id"`
	RPCURLs map[string]string `mapstructure:"rpc_urls"`
}

// Config holds all configuration
type Config struct {
	InterDSN string                           `mapstructure:"dsn"`
	Network  wtypes.Network                   `mapstructure:"network"`
	RPC      string                           `mapstructure:"rpc"`
	Protocol string                           `mapstructure:"protocol"`
	Location common.Location                  `mapstructure:"location"`
	KeyFile  string                           `mapstructure:"key_file"`
	Networks map[wtypes.Network]NetworkConfig `mapstructure:"networks"`
	Debug    bool                             `mapstructure:"debug"`
}

// LoadConfig loads configuration from config file
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	// If configPath is empty, look in default locations
	if configPath != "" {
		viper.AddConfigPath(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("$HOME/.quai-transfer")
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var rawConfig struct {
		InterDSN string `mapstructure:"dsn"`
		Network  string `mapstructure:"network"`
		Rpc      string `mapstructure:"rpc"`
		Protocol string `mapstructure:"protocol"`
		Location string `mapstructure:"location"`
		KeyFile  string `mapstructure:"key_file"`
		Networks map[string]struct {
			ChainID int64             `mapstructure:"chain_id"`
			RPCURLs map[string]string `mapstructure:"rpc_urls"`
		} `mapstructure:"networks"`
	}

	if err := viper.Unmarshal(&rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Convert the raw config to our Config type
	config := &Config{
		InterDSN: rawConfig.InterDSN,
		Network:  wtypes.Network(strings.ToLower(rawConfig.Network)),
		RPC:      rawConfig.Rpc,
		Protocol: rawConfig.Protocol,
		Location: stringToLocation(rawConfig.Location),
		KeyFile:  rawConfig.KeyFile,
		Networks: make(map[wtypes.Network]NetworkConfig),
	}

	// Validate network
	if !wtypes.ValidNetworks[config.Network] {
		return nil, fmt.Errorf("invalid network %q", config.Network)
	}

	// Convert networks map
	for name, netConfig := range rawConfig.Networks {
		network := wtypes.Network(strings.ToLower(name))
		if !wtypes.ValidNetworks[network] {
			return nil, fmt.Errorf("invalid network %q in networks configuration", name)
		}
		config.Networks[network] = NetworkConfig{
			ChainID: big.NewInt(netConfig.ChainID),
			RPCURLs: netConfig.RPCURLs,
		}
	}

	// Validate that the network exists in the Networks map
	if _, exists := config.Networks[config.Network]; !exists {
		return nil, fmt.Errorf("network %q configuration not found in networks section", config.Network)
	}

	GlobalLocation = config.Location
	return config, nil
}

func stringToLocation(s string) common.Location {
	var region, zone int
	fmt.Sscanf(s, "%d-%d", &region, &zone)
	loc, err := common.NewLocation(region, zone)
	if err != nil {
		panic(err)
	}
	return loc
}
