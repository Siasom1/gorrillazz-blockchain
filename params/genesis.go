package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type GenesisAlloc struct {
	Address common.Address
	Balance *big.Int
}

type GenesisConfig struct {
	ChainID uint64
	Alloc   []GenesisAlloc
}

// ----------------------------
// Create genesis configuration
// ----------------------------
func DefaultGenesis(gorrAdmin, usdccAdmin common.Address) *GenesisConfig {
	return &GenesisConfig{
		ChainID: 9999,
		Alloc: []GenesisAlloc{
			{
				Address: gorrAdmin,
				Balance: new(big.Int).Mul(big.NewInt(10_000_000_000), big.NewInt(1e18)),
			},
			{
				Address: usdccAdmin,
				Balance: new(big.Int).Mul(big.NewInt(10_000_000_000), big.NewInt(1e6)),
			},
		},
	}
}
