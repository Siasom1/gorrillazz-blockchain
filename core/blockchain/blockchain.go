package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/Siasom1/gorrillazz-chain/core/state"
	"github.com/Siasom1/gorrillazz-chain/core/txpool"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/data"
)

type Blockchain struct {
	dataDir   string
	networkID uint64

	head   *types.Block
	State  *state.StateDB
	TxPool *txpool.TxPool
}

func NewBlockchain(dataDir string, networkID uint64) (*Blockchain, error) {
	bc := &Blockchain{
		dataDir:   filepath.Join(dataDir, "chaindata"),
		networkID: networkID,
	}

	os.MkdirAll(bc.dataDir, 0o755)

	// Load state database
	st, err := state.NewStateDB(filepath.Join(dataDir, "state"))
	if err != nil {
		return nil, fmt.Errorf("state DB error: %v", err)
	}
	bc.State = st

	bc.TxPool = txpool.NewTxPool()

	// Try loading head
	head, err := bc.loadHead()
	if err != nil {
		return nil, err
	}

	if head != nil {
		fmt.Println("[CHAIN] Loaded existing chain head:", head.Header.Number)
		bc.head = head
		return bc, nil
	}

	// --------------------------------------
	// GENESIS CREATION
	// --------------------------------------
	fmt.Println("[GENESIS] No existing blockchain, creating new genesis block...")

	wallets, err := data.LoadOrCreateWallets(dataDir)
	if err != nil {
		return nil, err
	}

	admin := wallets.Admin.Address
	treasury := wallets.Treasury.Address

	fmt.Println("[GENESIS] Admin Wallet:", admin.Hex())
	fmt.Println("[GENESIS] Treasury Wallet:", treasury.Hex())

	// Alloc amounts
	adminAmount := new(big.Int).Mul(big.NewInt(10_000_000), big.NewInt(1e18))
	treasuryAmount := new(big.Int).Mul(big.NewInt(9_990_000_000), big.NewInt(1e18))

	// Apply to state
	bc.State.SetBalance(admin, adminAmount)
	bc.State.SetBalance(treasury, treasuryAmount)

	fmt.Println("[GENESIS] Alloc GORR → Admin:", adminAmount.String())
	fmt.Println("[GENESIS] Alloc GORR → Treasury:", treasuryAmount.String())

	// Build genesis block
	genesis := types.NewGenesisBlock()

	root, _ := bc.State.RootHash()
	genesis.Header.StateRoot = root

	bc.head = genesis
	bc.saveBlock(genesis)
	bc.saveHead()

	fmt.Println("[GENESIS] Genesis block created successfully!")

	return bc, nil
}

func (bc *Blockchain) NetworkID() uint64 {
	return bc.networkID
}

func (bc *Blockchain) Head() *types.Block {
	return bc.head
}

func (bc *Blockchain) saveHead() error {
	bytes, _ := json.MarshalIndent(bc.head, "", "  ")
	return os.WriteFile(filepath.Join(bc.dataDir, "head.json"), bytes, 0o644)
}

func (bc *Blockchain) loadHead() (*types.Block, error) {
	path := filepath.Join(bc.dataDir, "head.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	bytes, _ := os.ReadFile(path)
	var block types.Block
	if err := json.Unmarshal(bytes, &block); err != nil {
		return nil, err
	}
	return &block, nil
}

func (bc *Blockchain) saveBlock(block *types.Block) error {
	bytes, _ := json.MarshalIndent(block, "", "  ")
	return os.WriteFile(filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", block.Header.Number)), bytes, 0o644)
}
