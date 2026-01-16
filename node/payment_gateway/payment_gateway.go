package payment_gateway

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type PaymentStatus string

const (
	StatusPending PaymentStatus = "pending"
	StatusPaid    PaymentStatus = "paid"
	StatusRefund  PaymentStatus = "refunded"
)

type PaymentIntent struct {
	ID        uint64
	Merchant  common.Address
	Payer     common.Address
	Amount    *big.Int
	Token     string
	Status    PaymentStatus
	Timestamp uint64
}

type PaymentGateway struct {
	mu      sync.RWMutex
	counter uint64
	items   map[uint64]*PaymentIntent
}

func NewPaymentGateway() *PaymentGateway {
	return &PaymentGateway{
		items:   make(map[uint64]*PaymentIntent),
		counter: 1,
	}
}

func (pg *PaymentGateway) CreateIntent(merchant common.Address, amount *big.Int, token string, ts uint64) (*PaymentIntent, uint64, error) {
	if token != "GORR" && token != "USDCc" {
		return nil, 0, errors.New("unsupported token")
	}

	pg.mu.Lock()
	defer pg.mu.Unlock()

	id := pg.counter
	pg.counter++

	intent := &PaymentIntent{
		ID:        id,
		Merchant:  merchant,
		Amount:    amount,
		Token:     token,
		Status:    StatusPending,
		Timestamp: ts,
	}

	pg.items[id] = intent
	return intent, id, nil
}

func (pg *PaymentGateway) PayIntent(id uint64, payer common.Address) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	p, ok := pg.items[id]
	if !ok {
		return nil, fmt.Errorf("payment intent not found")
	}
	if p.Status != StatusPending {
		return nil, fmt.Errorf("invalid state: %s", p.Status)
	}
	p.Payer = payer
	p.Status = StatusPaid
	return p, nil
}

func (pg *PaymentGateway) RefundIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	p, ok := pg.items[id]
	if !ok {
		return nil, fmt.Errorf("payment intent not found")
	}
	if p.Status != StatusPaid {
		return nil, fmt.Errorf("not refundable state: %s", p.Status)
	}

	p.Status = StatusRefund
	return p, nil
}

func (pg *PaymentGateway) GetIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	p, ok := pg.items[id]
	if !ok {
		return nil, fmt.Errorf("payment intent not found")
	}
	return p, nil
}
