package rpc

import "math/big"

func calculateFee(amount *big.Int, bps uint64) *big.Int {
	if bps == 0 {
		return big.NewInt(0)
	}

	fee := new(big.Int).Mul(amount, big.NewInt(int64(bps)))
	fee.Div(fee, big.NewInt(10_000))
	return fee
}
