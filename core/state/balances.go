package state

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Balances struct {
	mu   sync.RWMutex
	gorr map[common.Address]*big.Int
	usdc map[common.Address]*big.Int
}

func NewBalances() *Balances {
	return &Balances{
		gorr: make(map[common.Address]*big.Int),
		usdc: make(map[common.Address]*big.Int),
	}
}

// -------- GORR --------

func (b *Balances) GetBalance(addr common.Address) *big.Int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if bal, ok := b.gorr[addr]; ok {
		return new(big.Int).Set(bal)
	}
	return big.NewInt(0)
}

func (b *Balances) AddBalance(addr common.Address, amount *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.gorr[addr]; !ok {
		b.gorr[addr] = big.NewInt(0)
	}
	b.gorr[addr].Add(b.gorr[addr], amount)
}

func (b *Balances) SubBalance(addr common.Address, amount *big.Int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.gorr[addr]; !ok {
		return false
	}
	if b.gorr[addr].Cmp(amount) < 0 {
		return false
	}
	b.gorr[addr].Sub(b.gorr[addr], amount)
	return true
}

// -------- USDCc --------

func (b *Balances) GetUSDCcBalance(addr common.Address) *big.Int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if bal, ok := b.usdc[addr]; ok {
		return new(big.Int).Set(bal)
	}
	return big.NewInt(0)
}

func (b *Balances) AddUSDCcBalance(addr common.Address, amount *big.Int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.usdc[addr]; !ok {
		b.usdc[addr] = big.NewInt(0)
	}
	b.usdc[addr].Add(b.usdc[addr], amount)
}

func (b *Balances) SubUSDCcBalance(addr common.Address, amount *big.Int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.usdc[addr]; !ok {
		return false
	}
	if b.usdc[addr].Cmp(amount) < 0 {
		return false
	}
	b.usdc[addr].Sub(b.usdc[addr], amount)
	return true
}
