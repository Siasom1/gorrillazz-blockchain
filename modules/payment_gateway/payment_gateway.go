package paymentgateway

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ---------------------------------------------
// Payment Status Types
// ---------------------------------------------

type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusPaid     PaymentStatus = "paid"
	StatusExpired  PaymentStatus = "expired"
	StatusRefunded PaymentStatus = "refunded"
	StatusSettled  PaymentStatus = "settled"
)

// ---------------------------------------------
// PaymentIntent struct
// ---------------------------------------------

type PaymentIntent struct {
	ID        uint64         `json:"ID"`
	Merchant  common.Address `json:"Merchant"`
	Payer     common.Address `json:"Payer"`
	Amount    *big.Int       `json:"Amount"`    // Brutobedrag dat de klant moet betalen (in wei)
	Token     string         `json:"Token"`     // "GORR" of "USDCc"
	Timestamp uint64         `json:"Timestamp"` // Aanmaak-tijd (unix seconds)
	Expiry    uint64         `json:"Expiry"`    // Verlooptijd (unix seconds)

	Paid     bool          `json:"Paid"`
	Refunded bool          `json:"Refunded"`
	Status   PaymentStatus `json:"Status"`

	// On-chain settlement metadata
	TxHash      string `json:"TxHash"`      // 0x-hash string
	BlockNumber uint64 `json:"BlockNumber"` // Block waar de betaling in zat
	PaidAt      uint64 `json:"PaidAt"`      // Block timestamp (unix) van betaling

	// Optioneel: later kun je hier fields toevoegen zoals:
	// MerchantNetAmount, TreasuryFeeAmount, Currency, Metadata, etc.
}

// ---------------------------------------------
// PaymentGateway struct
// ---------------------------------------------

type PaymentGateway struct {
	mu            sync.RWMutex
	intents       map[uint64]*PaymentIntent
	counter       uint64
	expirySeconds uint64 // standaard geldigheidsduur van een intent
}

// NewPaymentGateway maakt een nieuwe gateway.
// Standaard expiry: 15 minuten (900 seconden).
func NewPaymentGateway() *PaymentGateway {
	return &PaymentGateway{
		intents:       make(map[uint64]*PaymentIntent),
		counter:       0,
		expirySeconds: 900, // 15 min
	}
}

// Optioneel: runtime configuratie van expiry.
func (pg *PaymentGateway) SetExpirySeconds(seconds uint64) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.expirySeconds = seconds
}

// ---------------------------------------------
// Intent Lifecycle
// ---------------------------------------------

// CreateIntent maakt een nieuwe payment intent.
// merchant: merchant address
// amount:   brutobedrag dat klant moet betalen (wei)
// token:    "GORR" of "USDCc"
// ts:       unix timestamp (bijv. time.Now().Unix() of blockTime)
func (pg *PaymentGateway) CreateIntent(
	merchant common.Address,
	amount *big.Int,
	token string,
	ts uint64,
) (*PaymentIntent, uint64, error) {
	if amount == nil || amount.Sign() <= 0 {
		return nil, 0, errors.New("amount must be positive")
	}
	if token == "" {
		return nil, 0, errors.New("token is required")
	}

	pg.mu.Lock()
	defer pg.mu.Unlock()

	pg.counter++
	id := pg.counter

	intent := &PaymentIntent{
		ID:        id,
		Merchant:  merchant,
		Payer:     common.Address{},
		Amount:    new(big.Int).Set(amount),
		Token:     token,
		Timestamp: ts,
		Expiry:    ts + pg.expirySeconds,
		Paid:      false,
		Refunded:  false,
		Status:    StatusPending,
		TxHash:    "",
	}

	pg.intents[id] = intent
	return cloneIntent(intent), id, nil
}

// GetIntent haalt een intent op én update automatisch de status
// naar "expired" als de intent verlopen is en nog niet betaald.
func (pg *PaymentGateway) GetIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("payment intent not found")
	}

	now := uint64(time.Now().Unix())
	pg.updateStatusLocked(intent, now)

	return cloneIntent(intent), nil
}

// ListMerchantPayments geeft alle intents voor een merchant.
// Status wordt per intent bijgewerkt (expiry).
func (pg *PaymentGateway) ListMerchantPayments(merchant common.Address) []*PaymentIntent {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	now := uint64(time.Now().Unix())
	list := []*PaymentIntent{}

	for _, i := range pg.intents {
		if i.Merchant == merchant {
			pg.updateStatusLocked(i, now)
			list = append(list, cloneIntent(i))
		}
	}
	return list
}

// ---------------------------------------------
// "Soft" Pay / Refund API (RPC helpers)
// ---------------------------------------------

// PayIntent is een "zachte" betaalactie, zonder on-chain saldo checks.
// Handig voor tests / admin tooling, maar de ECHTE betaling
// hoort via MarkPaidFromTx + block producer te lopen.
func (pg *PaymentGateway) PayIntent(id uint64, payer common.Address) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("payment intent not found")
	}

	now := uint64(time.Now().Unix())
	pg.updateStatusLocked(intent, now)

	if intent.Status == StatusExpired {
		return nil, errors.New("cannot pay expired intent")
	}
	if intent.Status == StatusPaid || intent.Status == StatusRefunded || intent.Status == StatusSettled {
		return cloneIntent(intent), nil
	}

	intent.Paid = true
	intent.Status = StatusPaid
	intent.Payer = payer
	intent.PaidAt = now

	return cloneIntent(intent), nil
}

// RefundIntent markeert een intent als refunded.
// In de echte wereld zou hier ook saldo-verplaatsing bijkomen;
// hier markeren we alleen de intent state.
func (pg *PaymentGateway) RefundIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("payment intent not found")
	}

	now := uint64(time.Now().Unix())
	pg.updateStatusLocked(intent, now)

	if intent.Status != StatusPaid {
		return nil, errors.New("only paid intents can be refunded")
	}

	intent.Refunded = true
	intent.Status = StatusRefunded

	return cloneIntent(intent), nil
}

// Settlen (bijv. na uitbetaling naar bankrekening).
// Kan je later vanuit backend aanroepen.
func (pg *PaymentGateway) SettleIntent(id uint64) (*PaymentIntent, error) {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return nil, errors.New("payment intent not found")
	}

	if intent.Status != StatusPaid && intent.Status != StatusRefunded {
		return nil, errors.New("only paid or refunded intents can be settled")
	}

	intent.Status = StatusSettled
	return cloneIntent(intent), nil
}

// ---------------------------------------------
// On-chain settle: vanuit Block Producer
// ---------------------------------------------

// MarkPaidFromTx wordt in de BlockProducer aangeroepen zodra
// een payment-tx succesvol in een block komt.
//
// Hier doen we:
// - intent opzoeken
// - expiry check
// - merchant match
// - amount ≥ intent.Amount check
// - status = paid
// - on-chain metadata invullen (TxHash, BlockNumber, PaidAt)
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
		return errors.New("amount must be positive")
	}

	pg.mu.Lock()
	defer pg.mu.Unlock()

	intent, ok := pg.intents[id]
	if !ok {
		return errors.New("payment intent not found")
	}

	// Status updaten obv blockTime (chain time)
	pg.updateStatusLocked(intent, blockTime)

	if intent.Status == StatusExpired {
		return errors.New("intent is expired")
	}
	if intent.Status == StatusPaid || intent.Status == StatusRefunded || intent.Status == StatusSettled {
		return errors.New("intent already processed")
	}

	if merchant != intent.Merchant {
		return errors.New("merchant mismatch for intent")
	}

	// We eisen dat de on-chain betaalde waarde >= intent.Amount is
	if amount.Cmp(intent.Amount) < 0 {
		return errors.New("amount less than invoice value")
	}

	// Markeer als betaald
	intent.Paid = true
	intent.Status = StatusPaid
	intent.Payer = payer
	intent.PaidAt = blockTime
	intent.TxHash = txHash.Hex()
	intent.BlockNumber = blockNum

	return nil
}

// ---------------------------------------------
// Helpers
// ---------------------------------------------

// updateStatusLocked werkt onder pg.mu.Lock() / pg.mu.RLock().
// Als intent verlopen is en nog niet betaald -> StatusExpired.
func (pg *PaymentGateway) updateStatusLocked(intent *PaymentIntent, now uint64) {
	if intent.Status == StatusPending &&
		!intent.Paid &&
		!intent.Refunded &&
		intent.Expiry > 0 &&
		now > intent.Expiry {
		intent.Status = StatusExpired
	}
}

// cloneIntent maakt een kopie zodat de aanroeper de interne struct
// niet per ongeluk kan muteren.
func cloneIntent(i *PaymentIntent) *PaymentIntent {
	if i == nil {
		return nil
	}
	clone := *i
	if i.Amount != nil {
		clone.Amount = new(big.Int).Set(i.Amount)
	}
	return &clone
}
