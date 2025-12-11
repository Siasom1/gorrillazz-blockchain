package paymentgateway

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// Hoe lang een invoice geldig is (in seconden).
	// Nu: 1 uur. Later kun je dit eventueel configurabel maken.
	DefaultPaymentExpirySeconds = 3600
)

// PaymentIntent is een simpele in-memory representatie van een payment "invoice".
type PaymentIntent struct {
	ID        uint64         `json:"id"`
	Merchant  common.Address `json:"merchant"`
	Payer     common.Address `json:"payer"`
	Amount    *big.Int       `json:"amount"`
	Token     string         `json:"token"` // "GORR", "USDCc", etc.
	Timestamp uint64         `json:"timestamp"`
	ExpiresAt uint64         `json:"expiresAt"`
	Paid      bool           `json:"paid"`
	Refunded  bool           `json:"refunded"`

	// Extra info voor debug & tracking
	TxHash         common.Hash `json:"txHash"`
	ConfirmedBlock uint64      `json:"confirmedBlock"`
}

type PaymentGateway struct {
	mu      sync.RWMutex
	intents map[uint64]*PaymentIntent
	counter uint64
}

func NewPaymentGateway() *PaymentGateway {
	return &PaymentGateway{
		intents: make(map[uint64]*PaymentIntent),
		counter: 0,
	}
}

// CreateIntent maakt een nieuwe payment intent / invoice aan.
func (pg *PaymentGateway) CreateIntent(merchant common.Address, amount *big.Int, token string, ts uint64) (*PaymentIntent, uint64, error) {
	if amount == nil || amount.Sign() <= 0 {
		return nil, 0, errors.New("amount must be > 0")
	}
	if token == "" {
		return nil, 0, errors.New("missing token symbol")
	}

	pg.mu.Lock()
	defer pg.mu.Unlock()

	pg.counter++
	id := pg.counter

	intent := &PaymentIntent{
		ID:        id,
		Merchant:  merchant,
		Amount:    new(big.Int).Set(amount),
		Token:     token,
		Timestamp: ts,
		ExpiresAt: ts + DefaultPaymentExpirySeconds,
		Paid:      false,
		Refunded:  false,
	}

	pg.intents[id] = intent
	return intent, id, nil
}

// PayIntent (oude API) markeert een intent handmatig als betaald.
// Blijft bestaan voor admin / debugging, maar in jouw flow gebruiken
// we MarkPaidFromTx vanuit de BlockProducer.
func (pg *PaymentGateway) PayIntent(id uint64, payer common.Address) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("payment intent not found")
	}
	if intent.Refunded {
		return nil, errors.New("payment intent already refunded")
	}
	if intent.Paid {
		return intent, nil
	}

	intent.Paid = true
	intent.Payer = payer
	return intent, nil
}

// MarkPaidFromTx wordt aangeroepen door de BlockProducer zodra
// een payment tx in een blok is opgenomen.
//
// Hier gebeurt:
//   - check: bestaat intent?
//   - check: niet refunded / niet al betaald
//   - check: niet expired
//   - check: merchant & amount matchen
//   - intent.Paid = true, Payer, TxHash, ConfirmedBlock invullen
func (pg *PaymentGateway) MarkPaidFromTx(
	id uint64,
	payer common.Address,
	merchant common.Address,
	amount *big.Int,
	txHash common.Hash,
	blockNum uint64,
	blockTime uint64,
) error {
	if amount == nil || amount.Sign() <= 0 {
		return errors.New("invalid amount")
	}

	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return errors.New("payment intent not found")
	}

	if intent.Refunded {
		return errors.New("payment intent already refunded")
	}
	if intent.Paid {
		// al betaald, niets meer te doen
		return nil
	}

	// Expiry check
	if blockTime > intent.ExpiresAt {
		return errors.New("payment intent expired")
	}

	// Merchant moet kloppen
	if intent.Merchant != merchant {
		return errors.New("merchant mismatch for payment intent")
	}

	// Amount: on-chain bedrag moet minimaal de intent dekken.
	if amount.Cmp(intent.Amount) < 0 {
		return errors.New("on-chain amount is smaller than intent amount")
	}

	// Token-check: nu alleen "GORR", later uitbreiden naar USDCc etc.
	if intent.Token != "GORR" {
		// Je kunt dit strenger maken door een error te geven:
		// return errors.New("unsupported token for MarkPaidFromTx")
		// Voor nu laten we het toe als GORR-only chain.
	}

	intent.Paid = true
	intent.Payer = payer
	intent.TxHash = txHash
	intent.ConfirmedBlock = blockNum

	return nil
}

// RefundIntent markeert een intent als refunded (logisch / administratief).
func (pg *PaymentGateway) RefundIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("payment intent not found")
	}
	if !intent.Paid {
		return nil, errors.New("cannot refund — not yet paid")
	}
	if intent.Refunded {
		return intent, nil
	}

	intent.Refunded = true
	return intent, nil
}

func (pg *PaymentGateway) GetIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return intent, nil
}

func (pg *PaymentGateway) ListMerchantPayments(merchant common.Address) []*PaymentIntent {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	list := []*PaymentIntent{}
	for _, i := range pg.intents {
		if i.Merchant == merchant {
			list = append(list, i)
		}
	}
	return list
}
