package types

import (
	"github.com/ethereum/go-ethereum/common"
)

type Receipt struct {
	TxHash           common.Hash `json:"transactionHash"`
	BlockHash        common.Hash `json:"blockHash"`
	BlockNumber      uint64      `json:"blockNumber"`
	TransactionIndex uint64      `json:"transactionIndex"`

	From common.Address `json:"from"`
	To   common.Address `json:"to"`

	GasUsed uint64 `json:"gasUsed"`
	Status  uint64 `json:"status"` // 1 = success
}
