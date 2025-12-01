// core/types/block.go
package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
)

// Header beschrijft de metadata van een block
type Header struct {
	ParentHash common.Hash `json:"parentHash"`
	Number     uint64      `json:"number"`
	Time       uint64      `json:"timestamp"`
	StateRoot  common.Hash `json:"stateRoot"`
	TxRoot     common.Hash `json:"txRoot"`
}

// Block = header + lijst transacties
type Block struct {
	Header       *Header        `json:"header"`
	Transactions []*Transaction `json:"transactions"`
}

// SerializeHeader serialiseert alleen de header naar JSON (voor de hash)
func (b *Block) SerializeHeader() []byte {
	out, _ := json.Marshal(b.Header)
	return out
}

// Hash berekent de block hash op basis van de header
func (b *Block) Hash() common.Hash {
	return common.BytesToHash(Keccak256(b.SerializeHeader()))
}
