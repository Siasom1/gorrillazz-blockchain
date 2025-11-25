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
	dataDir := flag.String("datadir", "", "Data directory for blockchain data")
	netID := flag.Uint64("networkid", 9999, "Network ID")
	rpcPort := flag.Int("rpcport", 8545, "RPC port")
	logLevel := flag.String("loglevel", "info", "Log level: info/debug")

	flag.Parse()

	cfg := node.DefaultConfig()

	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	cfg.NetworkID = *netID
	cfg.RPCPort = *rpcPort
	cfg.LogLevel = *logLevel

	n := node.NewNode(cfg)

	if err := n.Start(); err != nil {
		fmt.Println("Error starting node:", err)
		return
	}

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
