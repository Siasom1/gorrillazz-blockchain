package node

import (
	"os"
)

type Config struct {
	DataDir   string
	NetworkID uint64
	RPCPort   int
	LogLevel  string
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		DataDir:   home + "/.gorrillazz",
		NetworkID: 9999,
		RPCPort:   8545,
		LogLevel:  "info",
	}
}
