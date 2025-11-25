package params

type ChainConfig struct {
	ChainID          uint64 `json:"chainId"`
	BlockTimeSeconds uint64 `json:"blockTimeSeconds"`
	GasLimit         uint64 `json:"gasLimit"`
}

func GorrillazzChainConfig() *ChainConfig {
	return &ChainConfig{
		ChainID:          9999,
		BlockTimeSeconds: 3,          // 3 seconden per block (aanpasbaar)
		GasLimit:         15_000_000, // placeholder
	}
}
