package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// ------------------------------------------------------------
// CHAIN CONFIG
// ------------------------------------------------------------

const (
	// ChainID van Gorrillazz Chain
	GorrChainID uint64 = 9999
)

// ------------------------------------------------------------
// SYSTEM WALLETS (VUL HIER JE EIGEN ADRESSEN IN)
// ------------------------------------------------------------

// LET OP: deze twee adressen komen uit jouw seed phrase
// (bijv. MetaMask of hardware wallet). Jij beheert de keys.
// VERVANG de placeholder strings hieronder door échte adressen.

var (
	AdminAddress    = common.HexToAddress("0x0000000000000000000000000000000000000000")
	TreasuryAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
)

// ------------------------------------------------------------
// NATIVE GORR SUPPLY
// ------------------------------------------------------------
//
// GORR is de native coin van de chain (zoals ETH op Ethereum)
// We nemen 18 decimalen, dus 1 GORR = 10^18 "wei".
//
// Totale supply: 100.000.000.000 GORR
// Admin:        10.000.000 GORR
// Treasury:     99.990.000.000 GORR
// ------------------------------------------------------------

var (
	// 10^18
	gorrDecimals = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)

	// 100.000.000.000 GORR * 1e18
	TotalGorrSupply = big.NewInt(0).Mul(
		big.NewInt(100_000_000_000),
		gorrDecimals,
	)

	// 10.000.000 GORR * 1e18
	AdminGorrAlloc = big.NewInt(0).Mul(
		big.NewInt(10_000_000),
		gorrDecimals,
	)

	// Treasury krijgt de rest
	TreasuryGorrAlloc = big.NewInt(0).Sub(
		TotalGorrSupply,
		AdminGorrAlloc,
	)
)

// ------------------------------------------------------------
// USDCc — systeemtoken
// ------------------------------------------------------------
//
// USDCc wordt later als systeemtoken / token-module geïmplementeerd.
// We reserveren hier alvast de totale supply configuratie.
// 6 decimalen zoals klassieke USDC.
// ------------------------------------------------------------

var (
	usdcDecimals = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(6), nil)

	// 100.000.000.000 USDCc * 1e6
	TotalUsdccSupply = big.NewInt(0).Mul(
		big.NewInt(100_000_000_000),
		usdcDecimals,
	)

	// 10.000.000 USDCc * 1e6
	AdminUsdccAlloc = big.NewInt(0).Mul(
		big.NewInt(10_000_000),
		usdcDecimals,
	)

	// Treasury krijgt de rest
	TreasuryUsdccAlloc = big.NewInt(0).Sub(
		TotalUsdccSupply,
		AdminUsdccAlloc,
	)
)

// ------------------------------------------------------------
// ROLES — voor AdminGuard, PaymentGateway, Bridge, etc.
// ------------------------------------------------------------

var (
	// Deze twee adressen zijn de "system owners"
	SystemAdminAddress    = AdminAddress
	SystemTreasuryAddress = TreasuryAddress
)
