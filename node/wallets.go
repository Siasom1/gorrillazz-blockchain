package node

import (
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
)

type WalletInfo struct {
	PrivateKey string `json:"privateKey"`
	Address    string `json:"address"`
}

type WalletSet struct {
	GORR  WalletInfo `json:"gorrAdmin"`
	USDCc WalletInfo `json:"usdccAdmin"`
}

func LoadOrCreateWallets(dataDir string) (*WalletSet, error) {
	path := filepath.Join(dataDir, "wallets.json")

	if _, err := os.Stat(path); err == nil {
		// Load existing
		bytes, _ := os.ReadFile(path)
		w := &WalletSet{}
		json.Unmarshal(bytes, w)
		return w, nil
	}

	// Generate new wallets
	gorrPriv, gorrAddr := newWallet()
	usdPriv, usdAddr := newWallet()

	wallets := &WalletSet{
		GORR: WalletInfo{
			PrivateKey: gorrPriv,
			Address:    gorrAddr,
		},
		USDCc: WalletInfo{
			PrivateKey: usdPriv,
			Address:    usdAddr,
		},
	}

	// Save
	out, _ := json.MarshalIndent(wallets, "", "  ")
	os.WriteFile(path, out, 0644)

	return wallets, nil
}

func newWallet() (string, string) {
	privateKey, _ := crypto.GenerateKey()
	privHex := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	keyBytes := crypto.FromECDSA(privateKey)
	keyHex := "0x" + (crypto.PubkeyToAddress(privateKey.PublicKey).Hex()[2:])

	return "0x" + string(crypto.FromECDSA(privateKey)), privHex
}

// Helper to convert private key to address
func AddressFromPrivate(priv *ecdsa.PrivateKey) string {
	return crypto.PubkeyToAddress(priv.PublicKey).Hex()
}
