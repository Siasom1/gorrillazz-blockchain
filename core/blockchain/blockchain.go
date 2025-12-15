package blockchain

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/txpool"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/state"

	payment_gateway "github.com/Siasom1/gorrillazz-chain/modules/payment_gateway"
	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

//
// --------------------------------------------------------
// Deterministic DEV wallets (Admin + Treasury)
// --------------------------------------------------------

const genesisSeedPhrase = "GORRILLAZZ DEV SEED PHRASE - CHANGE ME IN PRODUCTION"

type Wallet struct {
	Address    common.Address `json:"address"`
	PrivateKey string         `json:"privateKey"`
}

type WalletsFile struct {
	Admin    Wallet `json:"admin"`
	Treasury Wallet `json:"treasury"`
}

// Derive deterministic private key
func deriveKey(seed string, index uint32) (*ecdsa.PrivateKey, common.Address) {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", seed, index)))
	d := new(big.Int).SetBytes(hash[:])

	curve := gethcrypto.S256()
	nMinusOne := new(big.Int).Sub(curve.Params().N, big.NewInt(1))
	d.Mod(d, nMinusOne)
	d.Add(d, big.NewInt(1))

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: curve},
		D:         d,
	}
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())

	address := gethcrypto.PubkeyToAddress(priv.PublicKey)
	return priv, address
}

// Load or create wallets.json
func loadSystemWallets(datadir string) (WalletsFile, error) {
	path := filepath.Join(datadir, "wallets.json")

	// Exists? Load it.
	if _, err := os.Stat(path); err == nil {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return WalletsFile{}, err
		}
		var w WalletsFile
		if err := json.Unmarshal(bytes, &w); err != nil {
			return WalletsFile{}, err
		}
		return w, nil
	}

	// Otherwise generate deterministic wallets
	fmt.Println("[GENESIS] Creating Admin + Treasury wallets")

	adminPriv, adminAddr := deriveKey(genesisSeedPhrase, 0)
	trePriv, treAddr := deriveKey(genesisSeedPhrase, 1)

	w := WalletsFile{
		Admin: Wallet{
			Address:    adminAddr,
			PrivateKey: hex.EncodeToString(gethcrypto.FromECDSA(adminPriv)),
		},
		Treasury: Wallet{
			Address:    treAddr,
			PrivateKey: hex.EncodeToString(gethcrypto.FromECDSA(trePriv)),
		},
	}

	bytes, _ := json.MarshalIndent(w, "", "  ")
	_ = os.WriteFile(path, bytes, 0o600)

	return w, nil
}

//
// --------------------------------------------------------
// Blockchain Struct
// --------------------------------------------------------

type Blockchain struct {
	dataDir   string
	networkID uint64
	head      *types.Block
	State     *state.State
	TxPool    *txpool.TxPool
	Events    *events.EventBus

	// Payments / Gateway – same instance
	Payment *payment_gateway.PaymentGateway
	Gateway *payment_gateway.PaymentGateway

	AdminAddr    common.Address
	TreasuryAddr common.Address
}

//
// --------------------------------------------------------
// Constructor
// --------------------------------------------------------

func NewBlockchain(dataDir string, networkID uint64) (*Blockchain, error) {
	bc := &Blockchain{
		dataDir:   filepath.Join(dataDir, "chaindata"),
		Events:    events.NewEventBus(),
		networkID: networkID,
	}

	// Ensure dirs exist
	if err := os.MkdirAll(bc.dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %v", err)
	}

	// Load state DB
	st, err := state.NewState(filepath.Join(dataDir, "state"))
	if err != nil {
		return nil, fmt.Errorf("failed to load state db: %v", err)
	}
	bc.State = st

	// TxPool
	bc.TxPool = txpool.NewTxPool()

	// Payment gateway (one instance, two fields for compatibility)
	pg := payment_gateway.NewPaymentGateway()
	bc.Payment = pg
	bc.Gateway = pg

	// Load head
	head, err := bc.loadHead()
	if err != nil {
		return nil, err
	}

	// ========================================================
	// GENESIS
	// ========================================================
	if head == nil {
		fmt.Println("[GENESIS] No existing blockchain, creating genesis block...")

		wallets, err := loadSystemWallets(dataDir)
		if err != nil {
			return nil, fmt.Errorf("wallet load error: %w", err)
		}

		admin := wallets.Admin.Address
		treasury := wallets.Treasury.Address

		bc.AdminAddr = admin
		bc.TreasuryAddr = treasury

		fmt.Println("[GENESIS] Admin wallet:", admin.Hex())
		fmt.Println("[GENESIS] Treasury wallet:", treasury.Hex())

		// ---- ALL balances stored in WEI (1 token = 1e18) ----
		wei := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

		// Total supply: 100,000,000,000 GORR & 100,000,000,000 USDCc (in wei)
		totalGORR := new(big.Int).Mul(big.NewInt(100_000_000_000), wei)
		totalUSDCc := new(big.Int).Mul(big.NewInt(100_000_000_000), wei)

		// Admin gets 10,000,000 tokens (in wei)
		adminAlloc := new(big.Int).Mul(big.NewInt(10_000_000), wei)

		// Treasury gets the remainder (in wei)
		treasuryGORR := new(big.Int).Sub(totalGORR, adminAlloc)
		treasuryUSDCc := new(big.Int).Sub(totalUSDCc, adminAlloc)

		// Set balances
		bc.State.SetBalance(admin, adminAlloc)
		bc.State.SetBalance(treasury, treasuryGORR)

		bc.State.SetUSDCcBalance(admin, adminAlloc)
		bc.State.SetUSDCcBalance(treasury, treasuryUSDCc)

		genesis := &types.Block{
			Header: &types.Header{
				ParentHash: common.Hash{},
				Number:     0,
				Time:       uint64(time.Now().Unix()),
				StateRoot:  common.Hash{},
				TxRoot:     common.Hash{},
			},
			Transactions: []*types.Transaction{},
		}

		bc.head = genesis
		if err := bc.saveBlock(genesis); err != nil {
			return nil, err
		}
		if err := bc.saveHead(); err != nil {
			return nil, err
		}

		fmt.Println("[GENESIS] Genesis block #0 created.")
	} else {
		// Existing chain → reload wallets for AdminAddr/TreasuryAddr
		wallets, err := loadSystemWallets(dataDir)
		if err != nil {
			return nil, fmt.Errorf("wallet load error on existing chain: %w", err)
		}
		bc.AdminAddr = wallets.Admin.Address
		bc.TreasuryAddr = wallets.Treasury.Address
		bc.head = head
	}

	return bc, nil
}

//
// --------------------------------------------------------
// Misc functions
// --------------------------------------------------------

func (bc *Blockchain) NetworkID() uint64  { return bc.networkID }
func (bc *Blockchain) Head() *types.Block { return bc.head }

func (bc *Blockchain) SetHead(block *types.Block) error {
	bc.head = block
	if err := bc.saveBlock(block); err != nil {
		return err
	}
	return bc.saveHead()
}

func (bc *Blockchain) saveHead() error {
	path := filepath.Join(bc.dataDir, "head.json")
	data, err := json.MarshalIndent(bc.head, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (bc *Blockchain) loadHead() (*types.Block, error) {
	path := filepath.Join(bc.dataDir, "head.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b types.Block
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

//
// --------------------------------------------------------
// Block Storage
// --------------------------------------------------------

func (bc *Blockchain) saveBlock(block *types.Block) error {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", block.Header.Number))
	data, _ := json.MarshalIndent(block, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func (bc *Blockchain) LoadBlock(num uint64) (*types.Block, error) {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", num))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b types.Block
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

//
// --------------------------------------------------------
// Receipts + Tx Index
// --------------------------------------------------------

type txIndex map[string]uint64

func (bc *Blockchain) SaveReceipts(blockNum uint64, receipts []*types.Receipt) error {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("receipts_%d.json", blockNum))
	data, _ := json.MarshalIndent(receipts, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func (bc *Blockchain) LoadReceipts(blockNum uint64) ([]*types.Receipt, error) {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("receipts_%d.json", blockNum))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var receipts []*types.Receipt
	if err := json.Unmarshal(data, &receipts); err != nil {
		return nil, err
	}
	return receipts, nil
}

func (bc *Blockchain) loadTxIndex() (txIndex, error) {
	path := filepath.Join(bc.dataDir, "txindex.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return txIndex{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	idx := txIndex{}
	_ = json.Unmarshal(data, &idx)
	return idx, nil
}

func (bc *Blockchain) saveTxIndex(idx txIndex) error {
	path := filepath.Join(bc.dataDir, "txindex.json")
	data, _ := json.MarshalIndent(idx, "", "  ")
	return os.WriteFile(path, data, 0o644)
}

func (bc *Blockchain) SaveTxIndex(txHash common.Hash, blockNum uint64) error {
	idx, err := bc.loadTxIndex()
	if err != nil {
		return err
	}
	idx[txHash.Hex()] = blockNum
	return bc.saveTxIndex(idx)
}

func (bc *Blockchain) FindTxBlock(txHash common.Hash) (uint64, error) {
	idx, err := bc.loadTxIndex()
	if err != nil {
		return 0, err
	}
	num, ok := idx[txHash.Hex()]
	if !ok {
		return 0, fmt.Errorf("tx not indexed")
	}
	return num, nil
}
