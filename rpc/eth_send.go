package rpc

import (
	"fmt"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

func HandleSendTransaction(bc *blockchain.Blockchain, params []interface{}) (*types.Transaction, error) {
	if len(params) < 4 {
		return nil, fmt.Errorf("missing params: from, to, amount, token")
	}

	from := common.HexToAddress(params[0].(string))
	to := common.HexToAddress(params[1].(string))
	amount := new(big.Int)
	amount.SetString(params[2].(string), 10)
	token := params[3].(string)

	tx := &types.Transaction{
		From:  from,
		To:    &to,
		Value: amount,
		Token: token,
		Nonce: bc.State.GetNonce(from),
		Gas:   21000,
	}

	switch token {
	case "GORR":
		if err := bc.State.SubBalance(from, amount); err != nil {
			return nil, err
		}
		bc.State.AddBalance(to, amount)
	case "USDCc":
		if err := bc.State.SubUSDCc(from, amount); err != nil {
			return nil, err
		}
		bc.State.AddUSDCc(to, amount)
	default:
		return nil, fmt.Errorf("unknown token: %s", token)
	}

	bc.State.IncNonce(from)
	return tx, nil
}
