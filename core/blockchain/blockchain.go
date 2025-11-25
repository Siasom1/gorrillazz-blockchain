package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/Siasom1/gorrillazz-chain/core/txpool"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/state"

	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// -----------------------------------------------------------------------------
// Blockchain type
// -----------------------------------------------------------------------------

type Blockchain struct {
	dataDir   string
	networkID uint64

	head   *types.Block
	State  *state.State
	TxPool *txpool.TxPool
}

// -----------------------------------------------------------------------------
// Genesis wallet info (admin + treasury), opgeslagen in data/wallets.json
// -----------------------------------------------------------------------------

type WalletInfo struct {
	PrivateKey string `json:"privateKey"`
	Address    string `json:"address"`
}

type GenesisWallets struct {
	Admin    WalletInfo `json:"admin"`
	Treasury WalletInfo `json:"treasury"`
}

// -----------------------------------------------------------------------------
// NewBlockchain - constructor
// -----------------------------------------------------------------------------

func NewBlockchain(dataDir string, networkID uint64) (*Blockchain, error) {
	bc := &Blockchain{
		dataDir:   filepath.Join(dataDir, "chaindata"),
		networkID: networkID,
	}

	// Zorg dat chaindata map bestaat
	if err := os.MkdirAll(bc.dataDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create chaindata dir: %v", err)
	}

	// ----------------- STATE LADEN -----------------
	// State wordt opgeslagen in <datadir>/state (LevelDB)
	statePath := filepath.Join(dataDir, "state")
	st, err := state.NewState(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open state db: %v", err)
	}
	bc.State = st

	// ----------------- TX POOL -----------------
	bc.TxPool = txpool.NewTxPool()

	// ----------------- HEAD LADEN -----------------
	head, err := bc.loadHead()
	if err != nil {
		return nil, fmt.Errorf("failed to load head: %v", err)
	}

	if head == nil {
		// -------------------------------------------------
		// GEEN HEAD → NIEUWE GENESIS MAKEN
		// -------------------------------------------------
		if err := bc.initGenesis(dataDir); err != nil {
			return nil, fmt.Errorf("failed to init genesis: %v", err)
		}
	} else {
		// Bestaat al → gewoon gebruiken
		bc.head = head
	}

	return bc, nil
}

// -----------------------------------------------------------------------------
// initGenesis - maakt admin + treasury wallets en verdeelt GORR supply
// -----------------------------------------------------------------------------

func (bc *Blockchain) initGenesis(dataDir string) error {
	fmt.Println("[GENESIS] No existing blockchain, creating new genesis block...")

	// 1. Load of maak admin + treasury wallets
	wallets, err := loadOrCreateGenesisWallets(dataDir)
	if err != nil {
		return fmt.Errorf("failed to load/create genesis wallets: %v", err)
	}

	adminAddr := common.HexToAddress(wallets.Admin.Address)
	treasuryAddr := common.HexToAddress(wallets.Treasury.Address)

	fmt.Println("[GENESIS] Admin Wallet Address:   ", adminAddr.Hex())
	fmt.Println("[GENESIS] Treasury Wallet Address:", treasuryAddr.Hex())

	// 2. GORR supply:
	//    - Totaal: 10.000.000.000 GORR
	//    - Admin: 10.000.000 GORR
	//    - Treasury: rest (9.990.000.000 GORR)
	totalSupply := new(big.Int).Mul(big.NewInt(10_000_000_000), big.NewInt(1e18))
	adminAmount := new(big.Int).Mul(big.NewInt(10_000_000), big.NewInt(1e18))
	treasuryAmount := new(big.Int).Sub(totalSupply, adminAmount)

	fmt.Println("[GENESIS] Alloc GORR → Admin:   ", adminAmount.String())
	fmt.Println("[GENESIS] Alloc GORR → Treasury:", treasuryAmount.String())

	// 3. Apply allocs in state (native GORR balances)
	if err := bc.State.SetBalance(adminAddr, adminAmount); err != nil {
		return fmt.Errorf("failed to set admin GORR balance: %v", err)
	}
	if err := bc.State.SetBalance(treasuryAddr, treasuryAmount); err != nil {
		return fmt.Errorf("failed to set treasury GORR balance: %v", err)
	}

	// 4. Genesis block (native chain, GORR als enige native coin op dit niveau)
	genesis := types.NewGenesisBlock()
	// Voor nu laten we StateRoot/TxRoot op 0x00... (we kunnen later een echte trie toevoegen)

	bc.head = genesis

	// 5. Save genesis block + head pointer
	if err := bc.saveBlock(genesis); err != nil {
		return fmt.Errorf("failed to save genesis block: %v", err)
	}
	if err := bc.saveHead(); err != nil {
		return fmt.Errorf("failed to save head: %v", err)
	}

	fmt.Println("[GENESIS] Genesis block created successfully.")
	return nil
}

// -----------------------------------------------------------------------------
// Network ID
// -----------------------------------------------------------------------------

func (bc *Blockchain) NetworkID() uint64 {
	return bc.networkID
}

// -----------------------------------------------------------------------------
// HEAD functions
// -----------------------------------------------------------------------------

func (bc *Blockchain) Head() *types.Block {
	return bc.head
}

func (bc *Blockchain) SetHead(block *types.Block) error {
	bc.head = block

	// Save block N
	if err := bc.saveBlock(block); err != nil {
		return err
	}

	// Save head pointer
	return bc.saveHead()
}

func (bc *Blockchain) saveHead() error {
	path := filepath.Join(bc.dataDir, "head.json")

	bytes, err := json.MarshalIndent(bc.head, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, bytes, 0o644)
}

func (bc *Blockchain) loadHead() (*types.Block, error) {
	path := filepath.Join(bc.dataDir, "head.json")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b types.Block
	if err := json.Unmarshal(bytes, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

// -----------------------------------------------------------------------------
// Block Storage
// -----------------------------------------------------------------------------

func (bc *Blockchain) saveBlock(block *types.Block) error {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", block.Header.Number))

	bytes, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, bytes, 0o644)
}

func (bc *Blockchain) LoadBlock(number uint64) (*types.Block, error) {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("block_%d.json", number))

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b types.Block
	if err := json.Unmarshal(bytes, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

// -----------------------------------------------------------------------------
// Genesis Wallet Helpers
// -----------------------------------------------------------------------------

func loadOrCreateGenesisWallets(dataDir string) (*GenesisWallets, error) {
	path := filepath.Join(dataDir, "wallets.json")

	// Als bestand bestaat → inladen
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var w GenesisWallets
		if err := json.Unmarshal(data, &w); err != nil {
			return nil, err
		}
		return &w, nil
	}

	// Bestaat niet → nieuwe admin + treasury wallets genereren
	admin, err := generateWallet()
	if err != nil {
		return nil, err
	}
	treasury, err := generateWallet()
	if err != nil {
		return nil, err
	}

	w := &GenesisWallets{
		Admin:    admin,
		Treasury: treasury,
	}

	out, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, out, 0o644); err != nil {
		return nil, err
	}

	fmt.Println("[GENESIS] New wallets generated and saved to", path)
	fmt.Println("[GENESIS] Admin PrivateKey:   ", admin.PrivateKey)
	fmt.Println("[GENESIS] Treasury PrivateKey:", treasury.PrivateKey)

	return w, nil
}

func generateWallet() (WalletInfo, error) {
	privKey, err := gethcrypto.GenerateKey()
	if err != nil {
		return WalletInfo{}, err
	}

	privBytes := gethcrypto.FromECDSA(privKey)
	privHex := hex.EncodeToString(privBytes)

	address := gethcrypto.PubkeyToAddress(privKey.PublicKey)

	return WalletInfo{
		PrivateKey: "0x" + privHex,
		Address:    address.Hex(),
	}, nil
}
