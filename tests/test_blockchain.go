package main

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/payment_gateway"
	"github.com/Siasom1/gorrillazz-chain/core/producer"
	"github.com/Siasom1/gorrillazz-chain/core/state"
	"github.com/Siasom1/gorrillazz-chain/core/txpool"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	fmt.Println("=== Gorrillazz Test Script ===")

	// 1️⃣ Setup State + Blockchain
	st, _ := state.NewState("data")
	chain := &blockchain.Blockchain{
		State:        st,
		TxPool:       txpool.New(1000, 5*time.Second),
		TreasuryAddr: common.HexToAddress("0x2f74af61214e89796c37966d4b674a5ae148aa82"),
	}

	// 2️⃣ Admin wallet
	admin := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	st.SetBalance(admin, big.NewInt(10_000_000_000))     // 10B GORR
	st.SetUSDCcBalance(admin, big.NewInt(5_000_000_000)) // 5B USDCc

	// 3️⃣ Create Payment Gateway
	pg := payment_gateway.NewPaymentGateway()

	// 4️⃣ Create Wallets
	user := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	merchant := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")
	st.SetBalance(user, big.NewInt(1_000_000))
	st.SetUSDCcBalance(user, big.NewInt(500_000))

	// 5️⃣ Create Payment Intent
	amount := big.NewInt(100_000) // 100k GORR
	intent, id, _ := pg.CreateIntent(merchant, amount, "GORR", uint64(time.Now().Unix()))
	fmt.Printf("PaymentIntent created ID=%d | Amount=%s GORR\n", id, intent.Amount.String())

	// 6️⃣ Admin approves / PayIntent
	pg.PayIntent(id, admin)
	fmt.Printf("PaymentIntent ID=%d marked as paid by admin\n", id)

	// 7️⃣ Create transaction to merchant with PaymentIntent
	tx := &types.Transaction{
		From:  user,
		To:    &merchant,
		Value: amount,
		Token: "GORR",
		Data:  []byte(fmt.Sprintf("GORR_PAY:%d", id)),
		Gas:   21000,
	}

	// 8️⃣ Add to TxPool
	err := chain.TxPool.Add(tx)
	if err != nil {
		fmt.Println("TxPool.Add error:", err)
		return
	}
	fmt.Println("Transaction added to TxPool")

	// 9️⃣ Start BlockProducer
	log := &producer.BlockProducer{
		chain: chain,
		paygw: pg,
	}
	prod := producer.NewBlockProducer(chain, nil, 1, nil, pg)
	prod.produce() // produce a single block

	// 10️⃣ Check Balances
	userBal, _ := st.GetBalance(user)
	userUSDCc, _ := st.GetUSDCcBalance(user)
	merchantBal, _ := st.GetBalance(merchant)
	treasuryBal, _ := st.GetBalance(chain.TreasuryAddr)

	fmt.Printf("Balances after block:\n")
	fmt.Printf("User GORR: %s\n", userBal.String())
	fmt.Printf("User USDCc: %s\n", userUSDCc.String())
	fmt.Printf("Merchant GORR: %s\n", merchantBal.String())
	fmt.Printf("Treasury GORR: %s\n", treasuryBal.String())
}
