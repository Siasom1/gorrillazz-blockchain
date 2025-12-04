package node

// Config defines all runtime parameters used by the node.
type Config struct {
	DataDir   string
	RPCPort   int
	NetworkID uint64
	LogLevel  string
	BlockTime int // number of seconds between blocks
}

// DefaultConfig provides safe, working defaults.
func DefaultConfig() *Config {
	return &Config{
		DataDir:   "data",
		RPCPort:   9000,
		NetworkID: 9999,
		LogLevel:  "debug",
		BlockTime: 3, // block every 3 seconds
	}
}
