package node

import (
	"fmt"

	"github.com/Siasom1/gorrillazz-chain/consensus/producer"
	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/explorer"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/Siasom1/gorrillazz-chain/params"
	"github.com/Siasom1/gorrillazz-chain/rpc"
)

type Node struct {
	Config        *Config
	Logger        *log.Logger
	Chain         *blockchain.Blockchain
	BlockProducer *producer.BlockProducer
	RPCServer     *rpc.RPCServer
	ExplorerAPI   *explorer.ExplorerAPI
	Events        *events.EventBus
}

func NewNode(cfg *Config) *Node {
	logger := log.NewLogger(cfg.LogLevel)

	return &Node{
		Config: cfg,
		Logger: logger,
	}
}

func (n *Node) Start() error {
	n.Logger.Info("Starting Gorrillazz Node...")
	n.Logger.Info(fmt.Sprintf("Network ID: %d", n.Config.NetworkID))
	n.Logger.Info(fmt.Sprintf("Data directory: %s", n.Config.DataDir))
	n.Logger.Info(fmt.Sprintf("RPC port: %d", n.Config.RPCPort))

	// ------------------------------------------------
	// 1. EventBus
	// ------------------------------------------------
	n.Events = events.NewEventBus()

	// ------------------------------------------------
	// 2. Blockchain initialiseren
	// ------------------------------------------------
	bc, err := blockchain.NewBlockchain(n.Config.DataDir, n.Config.NetworkID)
	if err != nil {
		n.Logger.Error(fmt.Sprintf("Failed to init blockchain: %v", err))
		return err
	}
	n.Chain = bc

	head := n.Chain.Head()
	n.Logger.Info(fmt.Sprintf(
		"Loaded chain head: block #%d, hash=%s",
		head.Header.Number,
		head.Hash().String(),
	))

	// ------------------------------------------------
	// 3. Block Producer starten
	// ------------------------------------------------
	chainCfg := params.GorrillazzChainConfig()

	bp := producer.NewBlockProducer(
		n.Chain,
		n.Logger,
		chainCfg.BlockTimeSeconds,
		n.Events,
	)

	n.BlockProducer = bp
	n.BlockProducer.Start()

	// ------------------------------------------------
	// 4. RPC server starten
	// ------------------------------------------------
	handlers := rpc.NewHandlers(n.Chain)
	addr := fmt.Sprintf(":%d", n.Config.RPCPort)

	rpcServer := rpc.NewRPCServer(addr, handlers)
	n.RPCServer = rpcServer
	n.RPCServer.Start()

	n.Logger.Info("RPC server running on " + addr)

	// ------------------------------------------------
	// 5. Explorer API (REST + live streams) starten
	// ------------------------------------------------
	exp := explorer.NewExplorerAPI(n.Chain, n.Events)
	n.ExplorerAPI = exp
	// Explorer op :9500
	n.ExplorerAPI.Start(9500)

	n.Logger.Info("Explorer API running on :9500")
	n.Logger.Info("Node started successfully.")

	return nil
}

func (n *Node) Stop() {
	n.Logger.Info("Stopping Gorrillazz Node...")

	if n.BlockProducer != nil {
		n.BlockProducer.Stop()
	}
	if n.RPCServer != nil {
		n.RPCServer.Stop()
	}

	n.Logger.Info("Node stopped.")
}
