package blockchain

// ChainConfig defines how a blockchain instance should behave.
// This enables us to run multiple networks/binaries from the same codebase.
type ChainConfig struct {
	DataDir   string
	NetworkID uint64

	// Human label for the native currency of THIS chain.
	// NOTE: This is a label used by our node; MetaMask symbol is configured in the wallet UI.
	NativeSymbol string

	// Optional: different wallets seed per chain so addresses can differ.
	// If empty, default seed is used.
	GenesisSeed string
}

func DefaultChainConfig(dataDir string, networkID uint64) ChainConfig {
	return ChainConfig{
		DataDir:      dataDir,
		NetworkID:    networkID,
		NativeSymbol: "GORR",
		GenesisSeed:  "",
	}
}
