package rpc

import (
	"errors"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/ethereum/go-ethereum/common"
)

//
// --------------------------------------------------------
//  STRUCTURE VAN PARAMS
// --------------------------------------------------------

type SendTxParams struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

//
// --------------------------------------------------------
//  NATIVE GORR SEND
// --------------------------------------------------------

func HandleSendNative(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	to := common.HexToAddress(raw["to"].(string))
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))

	if err := bc.State.SubBalance(from, amount); err != nil {
		return nil, err
	}

	bc.State.AddBalance(to, amount)

	return map[string]interface{}{
		"success": true,
		"txHash":  "gorr_" + from.Hex(),
	}, nil
}

//
// --------------------------------------------------------
//  USDCc SEND
// --------------------------------------------------------

func HandleSendUSDCc(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if bc.State.Paused {
		return nil, errors.New("transfers paused")
	}

	raw := params[0].(map[string]interface{})
	from := common.HexToAddress(raw["from"].(string))
	to := common.HexToAddress(raw["to"].(string))
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))

	if err := bc.State.SubUSDCc(from, amount); err != nil {
		return nil, err
	}

	bc.State.AddUSDCc(to, amount)

	return map[string]interface{}{
		"success": true,
		"txHash":  "usdcc_" + from.Hex(),
	}, nil
}

//
// --------------------------------------------------------
//  GET BALANCES
// --------------------------------------------------------

func HandleGetBalance(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	addr := common.HexToAddress(params[0].(map[string]interface{})["address"].(string))
	bal, err := bc.State.GetBalance(addr)
	if err != nil {
		return nil, err
	}
	return map[string]string{"balance": bal.String()}, nil
}

func HandleGetUSDCcBalance(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	addr := common.HexToAddress(params[0].(map[string]interface{})["address"].(string))
	bal, err := bc.State.GetUSDCcBalance(addr)
	if err != nil {
		return nil, err
	}
	return map[string]string{"usdcc": bal.String()}, nil
}

func HandleGetSystemWallets(bc *blockchain.Blockchain, _ []interface{}) (interface{}, error) {
	return map[string]string{
		"admin":    bc.AdminAddr.Hex(),
		"treasury": bc.TreasuryAddr.Hex(),
	}, nil
}

//
// --------------------------------------------------------
//  ADMIN MINT / BURN (D.3)
// --------------------------------------------------------

func HandleAdminMint(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	to := common.HexToAddress(raw["to"].(string))
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))
	token := raw["token"].(string)

	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	switch token {
	case "GORR":
		bc.State.AddBalance(to, amount)
		bc.State.AddSupply("GORR", amount)
	case "USDCc":
		bc.State.AddUSDCc(to, amount)
		bc.State.AddSupply("USDCc", amount)
	default:
		return nil, errors.New("invalid token")
	}

	return map[string]interface{}{
		"success": true,
		"token":   token,
		"amount":  raw["amount"],
	}, nil
}

func HandleAdminBurn(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))
	token := raw["token"].(string)

	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	switch token {
	case "GORR":
		if err := bc.State.SubBalance(from, amount); err != nil {
			return nil, err
		}
		bc.State.SubSupply("GORR", amount)
	case "USDCc":
		if err := bc.State.SubUSDCc(from, amount); err != nil {
			return nil, err
		}
		bc.State.SubSupply("USDCc", amount)
	default:
		return nil, errors.New("invalid token")
	}

	return map[string]interface{}{
		"success": true,
		"token":   token,
		"amount":  raw["amount"],
	}, nil
}

func HandleAdminForceTransfer(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	target := common.HexToAddress(raw["target"].(string))
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))
	token := raw["token"].(string)

	// admin check
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	switch token {
	case "GORR":
		if err := bc.State.SubBalance(target, amount); err != nil {
			return nil, err
		}
		bc.State.AddBalance(bc.TreasuryAddr, amount)

	case "USDCc":
		if err := bc.State.SubUSDCc(target, amount); err != nil {
			return nil, err
		}
		bc.State.AddUSDCc(bc.TreasuryAddr, amount)

	default:
		return nil, errors.New("invalid token")
	}

	return map[string]any{
		"success": true,
		"from":    target.Hex(),
		"to":      bc.TreasuryAddr.Hex(),
		"amount":  raw["amount"],
		"token":   token,
	}, nil
}

//
// --------------------------------------------------------
//  ADMIN STATS (SAFE)
// --------------------------------------------------------

func HandleAdminStats(bc *blockchain.Blockchain, _ []interface{}) (interface{}, error) {
	return map[string]interface{}{
		"block":  bc.Head().Header.Number,
		"paused": bc.State.Paused,
		"supply": map[string]string{
			"GORR":  bc.State.GetTotalSupply("GORR").String(),
			"USDCc": bc.State.GetTotalSupply("USDCc").String(),
		},
		"treasury": bc.TreasuryAddr.Hex(),
	}, nil
}
