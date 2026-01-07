package rpc

import (
	"fmt"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/ethereum/go-ethereum/common"
)

// Mint tokens to any wallet
func HandleAdminMint(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("missing params: to, amount, token")
	}
	to := common.HexToAddress(params[0].(string))
	amount := new(big.Int)
	amount.SetString(params[1].(string), 10)
	token := params[2].(string)

	switch token {
	case "GORR":
		bc.State.AddBalance(to, amount)
	case "USDCc":
		bc.State.AddUSDCc(to, amount)
	default:
		return nil, fmt.Errorf("unknown token: %s", token)
	}

	return map[string]interface{}{
		"success": true,
		"to":      to.Hex(),
		"amount":  amount.String(),
		"token":   token,
	}, nil
}

// Burn tokens from any wallet
func HandleAdminBurn(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) < 3 {
		return nil, fmt.Errorf("missing params: from, amount, token")
	}
	from := common.HexToAddress(params[0].(string))
	amount := new(big.Int)
	amount.SetString(params[1].(string), 10)
	token := params[2].(string)

	switch token {
	case "GORR":
		if err := bc.State.SubBalance(from, amount); err != nil {
			return nil, err
		}
	case "USDCc":
		if err := bc.State.SubUSDCc(from, amount); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown token: %s", token)
	}

	return map[string]interface{}{
		"success": true,
		"from":    from.Hex(),
		"amount":  amount.String(),
		"token":   token,
	}, nil
}
