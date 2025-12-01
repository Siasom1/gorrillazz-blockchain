package node

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
)

type WalletFile struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"` // encrypted hex
}

// ------------------------------------------------------------
// AES-GCM encryptie helper
// ------------------------------------------------------------

func encrypt(data []byte, pass string) (string, error) {
	key := crypto.Keccak256([]byte(pass))

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	rand.Read(nonce)

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return hex.EncodeToString(ciphertext), nil
}

func decrypt(hexCipher string, pass string) ([]byte, error) {
	data, _ := hex.DecodeString(hexCipher)

	key := crypto.Keccak256([]byte(pass))

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce := data[:nonceSize]
	encrypted := data[nonceSize:]

	return gcm.Open(nil, nonce, encrypted, nil)
}

// ------------------------------------------------------------
// Create new wallet file
// ------------------------------------------------------------

func createWallet(path string, pass string) (*WalletFile, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	privBytes := crypto.FromECDSA(privateKey)
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	encrypted, err := encrypt(privBytes, pass)
	if err != nil {
		return nil, err
	}

	w := &WalletFile{
		Address:    address.Hex(),
		PrivateKey: encrypted,
	}

	data, _ := json.MarshalIndent(w, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}

	return w, nil
}

// ------------------------------------------------------------
// Load or create Admin & Treasury wallets
// ------------------------------------------------------------

type SystemWallets struct {
	Admin    *WalletFile
	Treasury *WalletFile
}

func LoadSystemWallets(datadir string) (*SystemWallets, error) {
	walletDir := filepath.Join(datadir, "wallets")
	os.MkdirAll(walletDir, os.ModePerm)

	adminPath := filepath.Join(walletDir, "admin.json")
	treasuryPath := filepath.Join(walletDir, "treasury.json")

	var adminWallet, treasuryWallet *WalletFile

	// -------- ADMIN --------
	if _, err := os.Stat(adminPath); os.IsNotExist(err) {
		fmt.Println("[WALLETS] Admin wallet missing -> generating new one...")
		adminWallet, err = createWallet(adminPath, "gorrillazz-admin")
		if err != nil {
			return nil, err
		}
		fmt.Println("[WALLETS] Admin Address:", adminWallet.Address)
	} else {
		data, _ := os.ReadFile(adminPath)
		json.Unmarshal(data, &adminWallet)
		fmt.Println("[WALLETS] Loaded Admin wallet:", adminWallet.Address)
	}

	// -------- TREASURY --------
	if _, err := os.Stat(treasuryPath); os.IsNotExist(err) {
		fmt.Println("[WALLETS] Treasury wallet missing -> generating new one...")
		treasuryWallet, err = createWallet(treasuryPath, "gorrillazz-treasury")
		if err != nil {
			return nil, err
		}
		fmt.Println("[WALLETS] Treasury Address:", treasuryWallet.Address)
	} else {
		data, _ := os.ReadFile(treasuryPath)
		json.Unmarshal(data, &treasuryWallet)
		fmt.Println("[WALLETS] Loaded Treasury wallet:", treasuryWallet.Address)
	}

	return &SystemWallets{
		Admin:    adminWallet,
		Treasury: treasuryWallet,
	}, nil
}
