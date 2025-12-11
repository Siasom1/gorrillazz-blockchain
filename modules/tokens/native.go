package tokens

import (
	"errors"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/state"
	"github.com/ethereum/go-ethereum/common"
)

type NativeTokenEngine struct {
	state        *state.State
	treasuryAddr common.Address
}

func NewNativeTokenEngine(st *state.State, treasury common.Address) *NativeTokenEngine {
	return &NativeTokenEngine{
		state:        st,
		treasuryAddr: treasury,
	}
}

func (n *NativeTokenEngine) TransferGORR(from, to common.Address, amount *big.Int) error {
	return n.transfer("GORR", from, to, amount)
}

func (n *NativeTokenEngine) TransferUSDCc(from, to common.Address, amount *big.Int) error {
	return n.transfer("USDCc", from, to, amount)
}

// -------------------------
// Mint/Burn (treasury only)
// -------------------------

func (n *NativeTokenEngine) MintGORR(to common.Address, amount *big.Int, caller common.Address) error {
	if caller != n.treasuryAddr {
		return errors.New("GOV: only treasury can mint GORR")
	}
	bal, _ := n.state.GetBalance(to)
	return n.state.SetBalance(to, new(big.Int).Add(bal, amount))
}

func (n *NativeTokenEngine) BurnGORR(from common.Address, amount *big.Int, caller common.Address) error {
	if caller != n.treasuryAddr {
		return errors.New("GOV: only treasury can burn GORR")
	}
	bal, _ := n.state.GetBalance(from)
	if bal.Cmp(amount) < 0 {
		return errors.New("burn exceeds balance")
	}
	return n.state.SetBalance(from, new(big.Int).Sub(bal, amount))
}

func (n *NativeTokenEngine) MintUSDCc(to common.Address, amount *big.Int, caller common.Address) error {
	if caller != n.treasuryAddr {
		return errors.New("GOV: only treasury can mint USDCc")
	}
	bal, _ := n.state.GetUSDCcBalance(to)
	return n.state.SetUSDCcBalance(to, new(big.Int).Add(bal, amount))
}

func (n *NativeTokenEngine) BurnUSDCc(from common.Address, amount *big.Int, caller common.Address) error {
	if caller != n.treasuryAddr {
		return errors.New("GOV: only treasury can burn USDCc")
	}
	bal, _ := n.state.GetUSDCcBalance(from)
	if bal.Cmp(amount) < 0 {
		return errors.New("burn exceeds balance")
	}
	return n.state.SetUSDCcBalance(from, new(big.Int).Sub(bal, amount))
}

// -------------------------
// INTERNAL TRANSFER
// -------------------------

func (n *NativeTokenEngine) transfer(token string, from, to common.Address, amount *big.Int) error {
	switch token {
	case "GORR":
		fromBal, _ := n.state.GetBalance(from)
		if fromBal.Cmp(amount) < 0 {
			return errors.New("insufficient GORR balance")
		}

		toBal, _ := n.state.GetBalance(to)

		n.state.SetBalance(from, new(big.Int).Sub(fromBal, amount))
		n.state.SetBalance(to, new(big.Int).Add(toBal, amount))
		return nil

	case "USDCc":
		fromBal, _ := n.state.GetUSDCcBalance(from)
		if fromBal.Cmp(amount) < 0 {
			return errors.New("insufficient USDCc balance")
		}

		toBal, _ := n.state.GetUSDCcBalance(to)

		n.state.SetUSDCcBalance(from, new(big.Int).Sub(fromBal, amount))
		n.state.SetUSDCcBalance(to, new(big.Int).Add(toBal, amount))
		return nil
	}

	return errors.New("unknown token")
}
