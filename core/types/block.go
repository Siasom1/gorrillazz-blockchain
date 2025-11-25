package types

import (
	"github.com/ethereum/go-ethereum/common"
)

// ------------------------------------------------------------
// Block Header
// ------------------------------------------------------------

type Header struct {
	ParentHash common.Hash `json:"parentHash"`
	Number     uint64      `json:"number"`
	Timestamp  uint64      `json:"timestamp"`
	StateRoot  common.Hash `json:"stateRoot"`
	TxRoot     common.Hash `json:"txRoot"`
}

// ------------------------------------------------------------
// Transaction (placeholder, expanded later)
// ------------------------------------------------------------

type Transaction struct {
	Nonce    uint64         `json:"nonce"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Value    uint64         `json:"value"`
	Data     []byte         `json:"data"`
	Hash     common.Hash    `json:"hash"`
	GasLimit uint64         `json:"gasLimit"`
	GasPrice uint64         `json:"gasPrice"`
}

// ------------------------------------------------------------
// Block
// ------------------------------------------------------------

type Block struct {
	Header       *Header        `json:"header"`
	Transactions []*Transaction `json:"transactions"`
}

// ------------------------------------------------------------
// NEW GENESIS BLOCK — ONLY HERE (remove genesis.go entirely)
// ------------------------------------------------------------

func NewGenesisBlock() *Block {
	header := &Header{
		ParentHash: common.Hash{}, // 0x000....
		Number:     0,
		Timestamp:  0,
		StateRoot:  common.Hash{}, // set later if you add real trie
		TxRoot:     common.Hash{},
	}

	return &Block{
		Header:       header,
		Transactions: []*Transaction{},
	}
}
