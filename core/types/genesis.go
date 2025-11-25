package types

import (
	"encoding/json"
	"math/big"

	// "time"

	"github.com/ethereum/go-ethereum/common"
)

type GenesisAlloc struct {
	Address common.Address
	Balance *big.Int
}

type GenesisData struct {
	Alloc []GenesisAlloc `json:"alloc"`
}

// func NewGenesisBlock() *Block {
// 	return &Block{
// 		Header: &Header{
// 			ParentHash: common.Hash{},
// 			Number:     0,
// 			Time:       uint64(time.Now().Unix()),
// 			StateRoot:  common.Hash{},
// 			TxRoot:     common.Hash{},
// 		},
// 		Transactions: []*Transaction{},
// 	}
// }

func NewGenesisAlloc(gorrAddr, usdAddr common.Address) *GenesisData {
	gorrAmount := new(big.Int).Mul(big.NewInt(10_000_000_000), big.NewInt(1e18))
	usdAmount := new(big.Int).Mul(big.NewInt(10_000_000_000), big.NewInt(1e6))

	return &GenesisData{
		Alloc: []GenesisAlloc{
			{Address: gorrAddr, Balance: gorrAmount},
			{Address: usdAddr, Balance: usdAmount},
		},
	}
}

func (g *GenesisData) JSON() []byte {
	out, _ := json.MarshalIndent(g, "", "  ")
	return out
}
