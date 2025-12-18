package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	rpcURL := "http://localhost:9000"

	// 🔐 ADMIN PRIVATE KEY (zonder 0x)
	privHex := "8685f1623ece272cde76efa71074da44d542e8036c2fe1e535ea6d1ee47987fd"

	// 🎯 Ontvanger (MetaMask)
	to := common.HexToAddress("0xPASTE_METAMASK_ADDRESS")

	// 💰 Bedrag: 1 GORR (1e18 wei)
	value := new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18))

	chainID := big.NewInt(9999) // jouw chainId

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA(privHex)
	if err != nil {
		log.Fatal(err)
	}

	from := crypto.PubkeyToAddress(privateKey.PublicKey)

	nonce, err := client.PendingNonceAt(context.Background(), from)
	if err != nil {
		log.Fatal(err)
	}

	gasLimit := uint64(21000)
	gasPrice := big.NewInt(0) // gas = 0 (dev mode)

	tx := types.NewTransaction(
		nonce,
		to,
		value,
		gasLimit,
		gasPrice,
		nil,
	)

	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Fatal(err)
	}

	raw, err := signedTx.MarshalBinary()
	if err != nil {
		log.Fatal(err)
	}

	rawHex := "0x" + hex.EncodeToString(raw)

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ TX SENT")
	fmt.Println("From:", from.Hex())
	fmt.Println("To:", to.Hex())
	fmt.Println("TxHash:", signedTx.Hash().Hex())
	fmt.Println("Raw:", rawHex)
}
