package rpc

import (
	"fmt"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

func (a *AdminAPI) Mint(
	to common.Address,
	amount *big.Int,
	token string,
) error {

	switch token {
	case types.TokenGORR:
		a.BC.State.AddBalance(to, amount)
	case types.TokenUSDCc:
		a.BC.State.AddUSDCcBalance(to, amount)
	default:
		return fmt.Errorf("unknown token")
	}

	return nil
}

func HandleAdminMint(bc *blockchain.Blockchain, to common.Address, amount *big.Int, token string) {
	switch token {
	case "GORR":
		bc.State.AddBalance(to, amount)
	case "USDCc":
		bc.State.AddUSDCc(to, amount)
	}
}
