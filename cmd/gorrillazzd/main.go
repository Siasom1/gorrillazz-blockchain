package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/Siasom1/gorrillazz-chain/node"
)

func main() {
	banner()

	// CLI flags
	dataDir := flag.String("datadir", "data", "Data directory for blockchain data")
	netID := flag.Uint64("networkid", 9999, "Network ID")
	rpcPort := flag.Int("rpcport", 9000, "RPC port")
	logLevel := flag.String("loglevel", "info", "Log level: info/debug")
	blockTime := flag.Int("blocktime", 3, "Block time in seconds")

	flag.Parse()

	cfg := node.DefaultConfig()

	// Override vanuit flags
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	cfg.NetworkID = *netID
	cfg.RPCPort = *rpcPort
	cfg.LogLevel = *logLevel
	cfg.BlockTime = *blockTime

	n, err := node.NewNode(cfg)
	if err != nil {
		fmt.Println("Error creating node:", err)
		return
	}

	if err := n.Start(); err != nil {
		fmt.Println("Error starting node:", err)
		return
	}

	// Process levend houden
	for {
		time.Sleep(1 * time.Second)
	}
}

func banner() {
	fmt.Println(`
   ██████╗  ██████╗ ██████╗ ██████╗ ███████╗██╗     ██╗     █████╗ ███████╗███████╗
  ██╔════╝ ██╔════╝██╔═══██╗██╔══██╗██╔════╝██║     ██║    ██╔══██╗██╔════╝██╔════╝
  ██║  ███╗██║     ██║   ██║██║  ██║█████╗  ██║     ██║    ███████║█████╗  ███████╗
  ██║   ██║██║     ██║   ██║██║  ██║██╔══╝  ██║     ██║    ██╔══██║██╔══╝  ╚════██║
  ╚██████╔╝╚██████╗╚██████╔╝██████╔╝███████╗███████╗███████╗██║  ██║███████╗███████║
   ╚═════╝  ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝
  `)
}
