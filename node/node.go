package node

import (
	"fmt"

	"github.com/Siasom1/gorrillazz-chain/consensus/producer"
	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/Siasom1/gorrillazz-chain/rpc"
)

type Node struct {
	Config   *Config
	Chain    *blockchain.Blockchain
	Logger   *log.Logger
	EventBus *events.EventBus
	Producer *producer.BlockProducer

	stopChan chan struct{}
}

// --------------------------------------------------------
// NewNode
// --------------------------------------------------------

func NewNode(cfg *Config) *Node {
	return &Node{
		Config:   cfg,
		Logger:   log.NewLogger(cfg.LogLevel),
		EventBus: events.NewEventBus(),
		stopChan: make(chan struct{}),
	}
}

// --------------------------------------------------------
// Start
// --------------------------------------------------------

func (n *Node) Start() error {
	n.Logger.Info("Starting Gorrillazz Node...")

	// Load blockchain
	chain, err := blockchain.NewBlockchain(n.Config.DataDir, n.Config.NetworkID)
	if err != nil {
		return fmt.Errorf("failed to init blockchain: %v", err)
	}
	n.Chain = chain

	// Start RPC server
	rpc.StartRPCServer(n.Config.RPCPort, n.Chain)

	// Start block producer
	n.Producer = producer.NewBlockProducer(
		n.Chain,
		n.Logger,
		uint64(n.Config.BlockTime),
		n.EventBus,
	)
	n.Producer.Start()

	n.Logger.Info("Node started successfully.")
	return nil
}

// --------------------------------------------------------
// Stop
// --------------------------------------------------------

func (n *Node) Stop() {
	n.Logger.Info("Stopping node...")
	close(n.stopChan)

	if n.Producer != nil {
		n.Producer.Stop()
	}

	n.Logger.Info("Node stopped.")
}
