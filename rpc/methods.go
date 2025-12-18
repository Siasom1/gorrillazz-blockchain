package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
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
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}
	raw := params[0].(map[string]interface{})

	// admin auth
	from := common.HexToAddress(raw["from"].(string))
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	target := common.HexToAddress(raw["target"].(string))

	token, _ := raw["token"].(string)
	if token == "" {
		token = "GORR"
	}

	amt, ok := raw["amount"]
	if !ok {
		return nil, errors.New("missing amount")
	}
	baseAmount, err := parseBigInt(amt) // accepts "100" or 100
	if err != nil || baseAmount.Sign() <= 0 {
		return nil, errors.New("invalid amount")
	}
	amountWei := new(big.Int).Mul(baseAmount, big.NewInt(1e18))

	to := bc.TreasuryAddr

	switch token {
	case "GORR":
		bal, err := bc.State.GetBalance(target)
		if err != nil {
			return nil, err
		}
		if bal.Cmp(amountWei) < 0 {
			return nil, errors.New("insufficient balance")
		}
		if err := bc.State.SubBalance(target, amountWei); err != nil {
			return nil, err
		}
		bc.State.AddBalance(to, amountWei)

		if target == (common.Address{}) {
			return nil, errors.New("invalid target")
		}

	case "USDCc":
		bal, err := bc.State.GetUSDCcBalance(target)
		if err != nil {
			return nil, err
		}
		if bal.Cmp(amountWei) < 0 {
			return nil, errors.New("insufficient balance")
		}
		if err := bc.State.SubUSDCc(target, amountWei); err != nil {
			return nil, err
		}
		bc.State.AddUSDCc(to, amountWei)

	default:
		return nil, errors.New("invalid token")
	}

	return map[string]any{
		"success": true,
		"token":   token,
		"from":    target.Hex(),
		"to":      to.Hex(),
		"amount":  baseAmount.String(), // human units
	}, nil
}

//
// --------------------------------------------------------
//  ADMIN STATS (SAFE)
// --------------------------------------------------------

func HandleAdminMintToTreasury(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	token := raw["token"].(string)
	amount := big.NewInt(int64(raw["amount"].(float64) * 1e18))

	switch token {
	case "GORR":
		bc.State.AddBalance(bc.TreasuryAddr, amount)
	case "USDCc":
		bc.State.AddUSDCc(bc.TreasuryAddr, amount)
	default:
		return nil, errors.New("invalid token")
	}

	return map[string]interface{}{
		"success":  true,
		"token":    token,
		"amount":   raw["amount"],
		"treasury": bc.TreasuryAddr.Hex(),
	}, nil
}

func HandleSendNative(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	to := common.HexToAddress(raw["to"].(string))
	// amount: number of string → *big.Int (human units)
	amt, ok := raw["amount"]
	if !ok {
		return nil, errors.New("missing amount")
	}

	baseAmount, err := parseBigInt(amt)
	if err != nil {
		return nil, errors.New("invalid amount")
	}

	// convert to wei (18 decimals)
	amount := new(big.Int).Mul(baseAmount, big.NewInt(1e18))

	// 2️⃣ fee calculation
	bps := bc.State.GetMerchantFeeBps()
	fee := calculateFee(amount, bps)
	net := new(big.Int).Sub(amount, fee)

	// 3️⃣ subtract full amount
	if err := bc.State.SubBalance(from, amount); err != nil {
		return nil, err
	}

	// 4️⃣ receiver gets net
	bc.State.AddBalance(to, net)

	// 5️⃣ treasury gets fee
	if fee.Sign() > 0 {
		bc.State.AddBalance(bc.TreasuryAddr, fee)
		bc.State.AddCollectedFee("GORR", fee)
	}

	if bc.State.Paused {
		return nil, errors.New("transfers paused")
	}

	return map[string]interface{}{
		"success": true,
		"from":    from.Hex(),
		"to":      to.Hex(),
		"net":     net.String(),
		"fee":     fee.String(),
	}, nil
}

func HandleSetFees(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	// merchantFeeBps moet number zijn (JSON number → float64)
	bpsFloat, ok := raw["merchantFeeBps"].(float64)
	if !ok {
		return nil, errors.New("invalid merchantFeeBps")
	}

	bps := uint64(bpsFloat)

	bc.State.SetMerchantFeeBps(bps)

	return map[string]interface{}{
		"success":        true,
		"merchantFeeBps": bps,
	}, nil
}

func HandleAdminStats(bc *blockchain.Blockchain, _ []interface{}) (interface{}, error) {
	treasury := bc.TreasuryAddr

	// balances
	gorrBal, _ := bc.State.GetBalance(treasury)
	usdccBal, _ := bc.State.GetUSDCcBalance(treasury)

	return map[string]interface{}{
		"block":          bc.Head().Header.Number,
		"paused":         bc.State.Paused,
		"treasury":       treasury.Hex(),
		"merchantFeeBps": bc.State.GetMerchantFeeBps(),

		"balances": map[string]string{
			"GORR":  gorrBal.String(),
			"USDCc": usdccBal.String(),
		},

		"supply": map[string]string{
			"GORR":  bc.State.GetTotalSupply("GORR").String(),
			"USDCc": bc.State.GetTotalSupply("USDCc").String(),
		},

		"fees": map[string]string{
			"GORR":  bc.State.GetCollectedFees("GORR").String(),
			"USDCc": bc.State.GetCollectedFees("USDCc").String(),
		},
	}, nil
}

func HandleAdminWithdrawFees(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}
	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	token := raw["token"].(string)
	if token == "" {
		token = "GORR"
	}

	// optional "to" (default admin)
	toAddr := bc.AdminAddr
	if v, ok := raw["to"].(string); ok && v != "" {
		toAddr = common.HexToAddress(v)
	}

	// amount is optional (human units). If missing => withdraw ALL collected fees (wei).
	collected := bc.State.GetCollectedFees(token)

	var withdrawWei *big.Int
	if v, ok := raw["amount"]; ok {
		amtHuman, err := parseBigInt(v) // whole units
		if err != nil {
			return nil, errors.New("invalid amount")
		}
		withdrawWei = new(big.Int).Mul(amtHuman, big.NewInt(1e18))
	} else {
		withdrawWei = new(big.Int).Set(collected)
	}

	if withdrawWei.Sign() <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	// enforce: cannot withdraw more than collected
	if collected.Cmp(withdrawWei) < 0 {
		return nil, errors.New("insufficient collected fees")
	}

	// move funds: treasury -> toAddr
	switch token {
	case "GORR":
		treBal, err := bc.State.GetBalance(bc.TreasuryAddr)
		if err != nil {
			return nil, err
		}
		if treBal.Cmp(withdrawWei) < 0 {
			return nil, errors.New("treasury insufficient balance")
		}
		if err := bc.State.SubBalance(bc.TreasuryAddr, withdrawWei); err != nil {
			return nil, err
		}
		bc.State.AddBalance(toAddr, withdrawWei)

	case "USDCc":
		treBal, err := bc.State.GetUSDCcBalance(bc.TreasuryAddr)
		if err != nil {
			return nil, err
		}
		if treBal.Cmp(withdrawWei) < 0 {
			return nil, errors.New("treasury insufficient balance")
		}
		if err := bc.State.SubUSDCc(bc.TreasuryAddr, withdrawWei); err != nil {
			return nil, err
		}
		bc.State.AddUSDCc(toAddr, withdrawWei)

	default:
		return nil, errors.New("invalid token")
	}

	_ = bc.State.SubCollectedFee(token, withdrawWei)

	return map[string]any{
		"success":     true,
		"token":       token,
		"to":          toAddr.Hex(),
		"withdrawWei": withdrawWei.String(),
	}, nil
}

func HandleAdminPauseTransfers(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing params")
	}
	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	paused, ok := raw["paused"].(bool)
	if !ok {
		return nil, errors.New("missing/invalid paused (bool)")
	}

	bc.State.Paused = paused

	return map[string]any{
		"success": true,
		"paused":  bc.State.Paused,
	}, nil
}

func parseBigInt(v interface{}) (*big.Int, error) {
	switch t := v.(type) {
	case float64:
		return big.NewInt(int64(t)), nil

	case string:
		if t == "" {
			return nil, fmt.Errorf("empty string")
		}
		n := new(big.Int)
		if _, ok := n.SetString(t, 10); !ok {
			return nil, fmt.Errorf("cannot parse %q", t)
		}
		return n, nil

	case json.Number:
		n := new(big.Int)
		if _, ok := n.SetString(t.String(), 10); !ok {
			return nil, fmt.Errorf("cannot parse %q", t.String())
		}
		return n, nil

	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

func HandlePauseTransfers(
	bc *blockchain.Blockchain,
	params []interface{},
) (interface{}, error) {

	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	raw := params[0].(map[string]interface{})

	from := common.HexToAddress(raw["from"].(string))
	paused := raw["paused"].(bool)

	// admin check
	if from != bc.AdminAddr {
		return nil, errors.New("admin only")
	}

	// state update
	bc.State.Paused = paused

	fmt.Println("🔥 EMIT PAUSE EVENT", paused)

	return map[string]any{
		"success": true,
		"paused":  paused,
	}, nil
}
