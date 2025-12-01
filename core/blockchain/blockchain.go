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
	"github.com/Siasom1/gorrillazz-chain/state"

	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// --------------------------------------------------------
// Config
// --------------------------------------------------------

// LET OP: deze seed is ALLEEN voor dev.
// Vervang dit in productie door een veilige seed of env-var.
const genesisSeedPhrase = "GORRILLAZZ DEV SEED PHRASE - CHANGE ME IN PRODUCTION"

type walletJSON struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"` // hex, zonder 0x
}

type genesisWallets struct {
	Admin    walletJSON `json:"admin"`
	Treasury walletJSON `json:"treasury"`
}

// --------------------------------------------------------
// Blockchain struct
// --------------------------------------------------------

type Blockchain struct {
	dataDir   string
	networkID uint64

	head   *types.Block
	State  *state.State
	TxPool *txpool.TxPool
}

// --------------------------------------------------------
// Helper: deterministische private key uit seed + index
// --------------------------------------------------------

func derivePrivKeyFromSeed(seed string, index uint32) (*ecdsa.PrivateKey, error) {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", seed, index)))
	d := new(big.Int).SetBytes(h[:])

	curve := gethcrypto.S256()
	nMinusOne := new(big.Int).Sub(curve.Params().N, big.NewInt(1))

	// 1 <= d < N
	d.Mod(d, nMinusOne)
	d.Add(d, big.NewInt(1))

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
		D: d,
	}
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())
	return priv, nil
}

func privKeyToHex(pk *ecdsa.PrivateKey) string {
	return hex.EncodeToString(gethcrypto.FromECDSA(pk))
}

// --------------------------------------------------------
// Constructor
// --------------------------------------------------------

func NewBlockchain(dataDir string, networkID uint64) (*Blockchain, error) {
	bc := &Blockchain{
		dataDir:   filepath.Join(dataDir, "chaindata"),
		networkID: networkID,
	}

	if err := os.MkdirAll(bc.dataDir, os.ModePerm); err != nil {
		return nil, err
	}

	// State DB
	st, err := state.NewState(filepath.Join(dataDir, "state"))
	if err != nil {
		return nil, fmt.Errorf("failed to open state db: %v", err)
	}
	bc.State = st

	// TxPool
	bc.TxPool = txpool.NewTxPool()

	// Head
	head, err := bc.loadHead()
	if err != nil {
		return nil, err
	}

	if head == nil {
		// ------------------------------------------------
		// FIRST START → GENESIS
		// ------------------------------------------------
		fmt.Println("[GENESIS] No existing blockchain, creating new genesis block...")

		// 1) Admin + Treasury keys/wallets (deterministisch uit seed)
		adminKey, err := derivePrivKeyFromSeed(genesisSeedPhrase, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to derive admin key: %w", err)
		}
		treasuryKey, err := derivePrivKeyFromSeed(genesisSeedPhrase, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to derive treasury key: %w", err)
		}

		adminAddr := gethcrypto.PubkeyToAddress(adminKey.PublicKey)
		treasuryAddr := gethcrypto.PubkeyToAddress(treasuryKey.PublicKey)

		fmt.Println("[GENESIS] Admin wallet address:   ", adminAddr.Hex())
		fmt.Println("[GENESIS] Treasury wallet address:", treasuryAddr.Hex())

		// 2) GORR alloc (native coin)
		//    1 GORR = 1e18 (zoals ETH)
		adminGORR := new(big.Int).Mul(big.NewInt(100000000000), big.NewInt(1e18))    // 100B GORR
		treasuryGORR := new(big.Int).Mul(big.NewInt(100000000000), big.NewInt(1e18)) // 100B GORR

		if err := bc.State.SetBalance(adminAddr, adminGORR); err != nil {
			return nil, fmt.Errorf("set admin balance: %w", err)
		}
		if err := bc.State.SetBalance(treasuryAddr, treasuryGORR); err != nil {
			return nil, fmt.Errorf("set treasury balance: %w", err)
		}

		// (USDCc als native systeemtokem komt later via tokens-module)

		// 3) Genesis block #0
		genesis := &types.Block{
			Header: &types.Header{
				ParentHash: common.Hash{},
				Number:     0,
				Time:       uint64(time.Now().Unix()),
				// StateRoot / TxRoot nog niet gevalideerd → voorlopig 0
				StateRoot: common.Hash{},
				TxRoot:    common.Hash{},
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

		fmt.Println("[GENESIS] Genesis block #0 created and saved.")

		// 4) wallets.json wegschrijven (zodat jij de keys hebt)
		rootDir := filepath.Dir(bc.dataDir) // bv. data/
		walletFile := filepath.Join(rootDir, "wallets.json")

		if _, err := os.Stat(walletFile); os.IsNotExist(err) {
			j := genesisWallets{
				Admin: walletJSON{
					Address:    adminAddr.Hex(),
					PrivateKey: privKeyToHex(adminKey),
				},
				Treasury: walletJSON{
					Address:    treasuryAddr.Hex(),
					PrivateKey: privKeyToHex(treasuryKey),
				},
			}

			data, err := json.MarshalIndent(j, "", "  ")
			if err == nil {
				_ = os.WriteFile(walletFile, data, 0o600)
				fmt.Println("[GENESIS] Wrote system wallets to", walletFile)
			}
		}
	} else {
		// Bestaande chain
		bc.head = head
	}

	return bc, nil
}

// --------------------------------------------------------
// Network ID
// --------------------------------------------------------

func (bc *Blockchain) NetworkID() uint64 {
	return bc.networkID
}

// --------------------------------------------------------
// Head helpers
// --------------------------------------------------------

func (bc *Blockchain) Head() *types.Block {
	return bc.head
}

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

// --------------------------------------------------------
// Block storage
// --------------------------------------------------------

func (bc *Blockchain) saveBlock(block *types.Block) error {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", block.Header.Number))

	data, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (bc *Blockchain) LoadBlock(number uint64) (*types.Block, error) {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", number))

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

// --------------------------------------------------------
// Receipts storage
// --------------------------------------------------------

func (bc *Blockchain) SaveReceipts(blockNum uint64, receipts []*types.Receipt) error {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("receipts_%d.json", blockNum))

	data, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		return err
	}

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

// --------------------------------------------------------
// Tx index: txHash -> blockNumber
// --------------------------------------------------------

type txIndex map[string]uint64

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
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	return idx, nil
}

func (bc *Blockchain) saveTxIndex(idx txIndex) error {
	path := filepath.Join(bc.dataDir, "txindex.json")

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}

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

	blockNum, ok := idx[txHash.Hex()]
	if !ok {
		return 0, fmt.Errorf("tx not indexed")
	}

	return blockNum, nil
}
