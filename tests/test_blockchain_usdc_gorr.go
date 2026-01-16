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
	fmt.Println("=== Gorrillazz Test Script: GORR + USDCc ===")

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

	// 3️⃣ Users / Merchants
	user := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	merchant := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")
	st.SetBalance(user, big.NewInt(1_000_000))
	st.SetUSDCcBalance(user, big.NewInt(500_000))

	// 4️⃣ Payment Gateway
	pg := payment_gateway.NewPaymentGateway()

	// 5️⃣ Create Payment Intents
	gorrAmount := big.NewInt(100_000) // GORR
	usdcAmount := big.NewInt(50_000)  // USDCc

	gorrIntent, gorrID, _ := pg.CreateIntent(merchant, gorrAmount, "GORR", uint64(time.Now().Unix()))
	usdcIntent, usdcID, _ := pg.CreateIntent(merchant, usdcAmount, "USDCc", uint64(time.Now().Unix()))

	fmt.Printf("PaymentIntent GORR ID=%d | Amount=%s\n", gorrID, gorrAmount.String())
	fmt.Printf("PaymentIntent USDCc ID=%d | Amount=%s\n", usdcID, usdcAmount.String())

	// 6️⃣ Admin approves
	pg.PayIntent(gorrID, admin)
	pg.PayIntent(usdcID, admin)

	// 7️⃣ Create transactions
	txGORR := &types.Transaction{
		From:  user,
		To:    &merchant,
		Value: gorrAmount,
		Token: "GORR",
		Data:  []byte(fmt.Sprintf("GORR_PAY:%d", gorrID)),
		Gas:   21000,
	}

	txUSDC := &types.Transaction{
		From:  user,
		To:    &merchant,
		Value: usdcAmount,
		Token: "USDCc",
		Data:  []byte(fmt.Sprintf("GORR_PAY:%d", usdcID)),
		Gas:   21000,
	}

	// 8️⃣ Add to TxPool
	if err := chain.TxPool.Add(txGORR); err != nil {
		fmt.Println("TxPool.Add GORR error:", err)
	}
	if err := chain.TxPool.Add(txUSDC); err != nil {
		fmt.Println("TxPool.Add USDCc error:", err)
	}
	fmt.Println("Transactions added to TxPool")

	// 9️⃣ Produce a block
	prod := producer.NewBlockProducer(chain, nil, 1, nil)
	prod.produce()

	// 🔟 Check balances
	userGORR, _ := st.GetBalance(user)
	userUSDC, _ := st.GetUSDCcBalance(user)
	merchantGORR, _ := st.GetBalance(merchant)
	merchantUSDC, _ := st.GetUSDCcBalance(merchant)
	treasuryGORR, _ := st.GetBalance(chain.TreasuryAddr)
	treasuryUSDC, _ := st.GetUSDCcBalance(chain.TreasuryAddr)

	fmt.Println("=== Balances after block ===")
	fmt.Printf("User GORR: %s\n", userGORR.String())
	fmt.Printf("User USDCc: %s\n", userUSDC.String())
	fmt.Printf("Merchant GORR: %s\n", merchantGORR.String())
	fmt.Printf("Merchant USDCc: %s\n", merchantUSDC.String())
	fmt.Printf("Treasury GORR: %s\n", treasuryGORR.String())
	fmt.Printf("Treasury USDCc: %s\n", treasuryUSDC.String())
}
