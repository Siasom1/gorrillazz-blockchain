package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/Siasom1/gorrillazz-chain/rpc"
)

func main() {
	fmt.Println("[USDCcD] Starting USDCc-native chain...")

	// CLI flags
	dataDir := flag.String("datadir", "data-usdcc", "USDCc data directory")
	rpcPort := flag.Int("rpcport", 9100, "RPC port")
	logLevel := flag.String("loglevel", "info", "Log level: info/debug")

	flag.Parse()

	logger := log.NewLogger(*logLevel)

	// Event bus
	bus := events.NewEventBus()

	// Chain config — USDCc is native
	cfg := blockchain.ChainConfig{
		DataDir:      *dataDir,
		NetworkID:    9998, // HARD-CODED chainId for USDCc chain
		NativeSymbol: "USDCc",
	}

	chain, err := blockchain.NewBlockchainWithConfig(cfg)
	if err != nil {
		fmt.Println("Error creating USDCc chain:", err)
		return
	}

	rpcServer := rpc.NewServer(chain, bus)

	go rpc.StartRPCServer(*rpcPort, rpcServer)

	logger.Info("USDCc chain started successfully.")

	for {
		time.Sleep(1 * time.Second)
	}
}
