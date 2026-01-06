package rpc

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

func (e *ethRPC) Call(to *common.Address, data []byte, caller common.Address) ([]byte, error) {
	if to == nil {
		return nil, fmt.Errorf("missing to")
	}

	// Only USDCc pseudo-contract
	if *to != USDCcContract {
		return nil, nil
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("invalid call data")
	}

	methodID := hex.EncodeToString(data[:4])

	// balanceOf(address)
	if methodID == "70a08231" {
		addr := common.BytesToAddress(data[16:36])
		bal := e.bc.State.GetUSDCcBalance(addr)
		return common.LeftPadBytes(bal.Bytes(), 32), nil
	}

	// transfer(address,uint256)
	if methodID == "a9059cbb" {
		to := common.BytesToAddress(data[16:36])
		amount := new(big.Int).SetBytes(data[36:68])

		tx := &types.Transaction{
			From:  caller,
			To:    &to,
			Value: amount,
			Token: types.TokenUSDCc,
		}

		return nil, e.bc.ProcessTransaction(tx)
	}

	return nil, fmt.Errorf("unknown method")
}
