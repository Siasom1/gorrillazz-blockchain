package node

import (
	"fmt"

	"github.com/Siasom1/gorrillazz-chain/consensus/producer"
	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/Siasom1/gorrillazz-chain/rpc"
)

// Node is de “container” voor alles:
// - blockchain
// - event bus
// - block producer
// - RPC server
type Node struct {
	Config *Config

	Chain  *blockchain.Blockchain
	Logger *log.Logger
	Bus    *events.EventBus

	Producer *producer.BlockProducer
	RPC      *rpc.Server

	stopChan chan struct{}
}

// NewNode bouwt alle componenten op
func NewNode(cfg *Config) (*Node, error) {
	logger := log.NewLogger(cfg.LogLevel)

	// Shared event bus
	bus := events.NewEventBus()

	// Blockchain
	chain, err := blockchain.NewBlockchain(cfg.DataDir, cfg.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("init blockchain: %w", err)
	}

	// Block producer
	chain.Events = bus
	prod := producer.NewBlockProducer(
		chain,
		logger,
		uint64(cfg.BlockTime),
		bus,
	)

	// RPC server
	rpcServer := rpc.NewServer(chain, bus)

	return &Node{
		Config:   cfg,
		Chain:    chain,
		Logger:   logger,
		Bus:      bus,
		Producer: prod,
		RPC:      rpcServer,
		stopChan: make(chan struct{}),
	}, nil
}

// Start start de RPC-server + block producer
func (n *Node) Start() error {
	n.Logger.Info("Starting Gorrillazz Node...")

	// Start RPC in een goroutine
	go rpc.StartRPCServer(n.Config.RPCPort, n.RPC)

	// Start block producer
	n.Producer.Start()

	n.Logger.Info("Node started successfully.")
	return nil
}

// Stop netjes stoppen
func (n *Node) Stop() {
	n.Logger.Info("Stopping node...")
	close(n.stopChan)

	if n.Producer != nil {
		n.Producer.Stop()
	}

	n.Logger.Info("Node stopped.")
}
