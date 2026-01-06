package admin

import (
	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
)

type AdminAPI struct {
	BC *blockchain.Blockchain
}

func NewAdminAPI(bc *blockchain.Blockchain) *AdminAPI {
	return &AdminAPI{BC: bc}
}
