package main

import (
	"fmt"
	"math/big"
	"time"

	"gorrillazz-chain/core/blockchain"
	"gorrillazz-chain/core/types"
	"gorrillazz-chain/events"
	"gorrillazz-chain/log"
	"gorrillazz-chain/producer"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	fmt.Println("Stap 7 test gestart...")

	// Logger en EventBus
	logger := log.NewLogger()
	bus := events.NewEventBus()

	// Maak blockchain aan
	chain, err := blockchain.NewBlockchain("state_db")
	if err != nil {
		logger.Error(fmt.Sprintf("Blockchain init failed: %v", err))
		return
	}
	defer chain.State.Close()

	// Admin adres instellen
	admin := common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	chain.AdminAddr = admin
	chain.TreasuryAddr = common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB")

	// Voeg genesis block toe
	genesis := &types.Block{
		Header: &types.Header{
			Number: 0,
			Time:   uint64(time.Now().Unix()),
		},
		Transactions: []*types.Transaction{},
	}
	if err := chain.SetHead(genesis); err != nil {
		logger.Error(fmt.Sprintf("SetHead genesis failed: %v", err))
		return
	}

	// Voeg test saldo toe
	chain.State.SetBalance(admin, big.NewInt(1_000_000))

	// Start BlockProducer
	bp := producer.NewBlockProducer(chain, logger, 1, bus)
	bp.Start()
	defer bp.Stop()

	// Maak een testtransactie
	to := common.HexToAddress("0xCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC")
	tx := &types.Transaction{
		FromAddr: admin,
		To:       &to,
		Value:    big.NewInt(500),
		Nonce:    0,
		Token:    types.TokenGORR,
		Gas:      0,
	}

	// Voeg transaction toe aan pool
	chain.TxPool.Add(tx)

	// Wacht een paar seconden zodat de producer blok kan maken
	time.Sleep(2 * time.Second)

	// Check balances
	fromBal, _ := chain.State.GetBalance(admin)
	toBal, _ := chain.State.GetBalance(to)
	fmt.Printf("Admin balance: %s\n", fromBal.String())
	fmt.Printf("To balance: %s\n", toBal.String())

	fmt.Println("Stap 7 test klaar.")
}
