package types

import (
	"github.com/ethereum/go-ethereum/common"
)

type Receipt struct {
	TxHash           common.Hash `json:"transactionHash"`
	BlockHash        common.Hash `json:"blockHash"`
	BlockNumber      uint64      `json:"blockNumber"`
	TransactionIndex uint64      `json:"transactionIndex"`
	Status           uint64      `json:"status"`
	GasUsed          uint64      `json:"gasUsed"`
	Logs             []interface{}
	From             common.Address `json:"from"`
	To               common.Address `json:"to"`
}
