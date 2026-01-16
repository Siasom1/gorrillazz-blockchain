package blockchain

import (
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

var Treasury = common.HexToAddress("0x2f74af61214e89796c37966d4b674a5ae148aa82")

func (bc *Blockchain) ApplyTransaction(tx *types.Transaction) bool {
	from := tx.From
	to := tx.To

	if to == (common.Address{}) {
		return false
	}

	val := new(big.Int).SetUint64(tx.Value)

	switch tx.Token {
	case types.TokenGORR:
		return bc.processGORR(from, to, val)
	case types.TokenUSDCC:
		return bc.processUSDCc(from, to, val)
	default:
		return false
	}
}

func (bc *Blockchain) processGORR(from, to common.Address, val *big.Int) bool {
	fromBal, _ := bc.State.GetBalance(from)
	if fromBal.Cmp(val) < 0 {
		return false
	}

	toBal, _ := bc.State.GetBalance(to)

	bc.State.SetBalance(from, new(big.Int).Sub(fromBal, val))
	bc.State.SetBalance(to, new(big.Int).Add(toBal, val))
	return true
}

func (bc *Blockchain) processUSDCc(from, to common.Address, val *big.Int) bool {
	fromBal, _ := bc.State.GetUSDCcBalance(from)
	if fromBal.Cmp(val) < 0 {
		return false
	}

	fee := new(big.Int).Mul(val, big.NewInt(250))
	fee.Div(fee, big.NewInt(10000))

	net := new(big.Int).Sub(val, fee)

	toBal, _ := bc.State.GetUSDCcBalance(to)
	treasuryBal, _ := bc.State.GetUSDCcBalance(Treasury)

	bc.State.SetUSDCcBalance(from, new(big.Int).Sub(fromBal, val))
	bc.State.SetUSDCcBalance(to, new(big.Int).Add(toBal, net))
	bc.State.SetUSDCcBalance(Treasury, new(big.Int).Add(treasuryBal, fee))

	return true
}
