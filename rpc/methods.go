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
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	// Decode JSON → struct
	raw := params[0].(map[string]interface{})

	p := SendTxParams{
		From:   raw["from"].(string),
		To:     raw["to"].(string),
		Amount: raw["amount"].(float64),
	}

	from := common.HexToAddress(p.From)
	to := common.HexToAddress(p.To)

	amount := big.NewInt(int64(p.Amount * 1e18))

	// balance checks
	_, err := bc.State.GetBalance(from)
	if err != nil {
		return nil, err
	}

	// subtract + add
	if err := bc.State.SubBalance(from, amount); err != nil {
		return nil, err
	}

	bc.State.AddBalance(to, amount)

	return map[string]interface{}{
		"success": true,
		"txHash":  "gorr_native_" + from.Hex(),
	}, nil
}

//
// --------------------------------------------------------
//  USDCc SEND
// --------------------------------------------------------

func HandleSendUSDCc(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	raw := params[0].(map[string]interface{})
	p := SendTxParams{
		From:   raw["from"].(string),
		To:     raw["to"].(string),
		Amount: raw["amount"].(float64),
	}

	from := common.HexToAddress(p.From)
	to := common.HexToAddress(p.To)

	amount := big.NewInt(int64(p.Amount * 1e18))

	_, err := bc.State.GetUSDCcBalance(from)
	if err != nil {
		return nil, err
	}

	if err := bc.State.SubUSDCc(from, amount); err != nil {
		return nil, err
	}

	bc.State.AddUSDCc(to, amount)

	return map[string]interface{}{
		"success": true,
		"txHash":  "usdcc_tx_" + from.Hex(),
	}, nil
}

//
// --------------------------------------------------------
//  GET BALANCES
// --------------------------------------------------------

func HandleGetBalance(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})
	addr := common.HexToAddress(raw["address"].(string))

	bal, err := bc.State.GetBalance(addr)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"balance": bal.String(),
	}, nil
}

func HandleGetUSDCcBalance(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})
	addr := common.HexToAddress(raw["address"].(string))

	bal, err := bc.State.GetUSDCcBalance(addr)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"usdcc": bal.String(),
	}, nil
}

func HandleGetSystemWallets(bc *blockchain.Blockchain, _ []interface{}) (interface{}, error) {
	return map[string]string{
		"admin":    bc.AdminAddr.Hex(),
		"treasury": bc.TreasuryAddr.Hex(),
	}, nil
}

func HandleAdminMint(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})

	to := common.HexToAddress(raw["to"].(string))
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))
	token := raw["token"].(string)

	if common.HexToAddress(raw["from"].(string)) != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	switch token {
	case "GORR":
		bc.State.AddBalance(to, amount)
	case "USDCc":
		bc.State.AddUSDCc(to, amount)
	default:
		return nil, errors.New("invalid token")
	}

	return map[string]any{
		"success": true,
		"to":      to.Hex(),
		"amount":  raw["amount"],
		"token":   token,
	}, nil
}

func HandleGetAllBalances(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})
	addr := common.HexToAddress(raw["address"].(string))

	gorr, err := bc.State.GetBalance(addr)
	if err != nil {
		return nil, err
	}

	usdcc, err := bc.State.GetUSDCcBalance(addr)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"GORR":  gorr.String(),
		"USDCc": usdcc.String(),
	}, nil
}
