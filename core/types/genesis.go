package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type GenesisConfig struct {
	ChainID  uint64 `json:"chainId"`
	Name     string `json:"name"`
	Admin    string `json:"admin"`
	Treasury string `json:"treasury"`
}

// BuildGenesisBlock maakt het genesis block
func BuildGenesisBlock(cfg GenesisConfig) *Block {
	// admin & treasury bewust ingelezen (later nodig voor mint)
	admin := common.HexToAddress(cfg.Admin)
	treasury := common.HexToAddress(cfg.Treasury)

	// voorkom unused errors (voor nu)
	_ = admin
	_ = treasury

	header := &Header{
		Number: 0,
		Time:   uint64(time.Now().Unix()),
	}

	block := &Block{
		Header:       header,
		Transactions: []*Transaction{},
	}

	return block
}
