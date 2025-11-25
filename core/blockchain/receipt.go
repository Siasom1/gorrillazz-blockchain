package blockchain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

func (bc *Blockchain) SaveReceipts(blockNum uint64, receipts []*types.Receipt) error {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("receipts_%d.json", blockNum))
	bytes, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o644)
}

func (bc *Blockchain) LoadReceipts(blockNum uint64) ([]*types.Receipt, error) {
	path := filepath.Join(bc.dataDir, fmt.Sprintf("receipts_%d.json", blockNum))

	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var receipts []*types.Receipt
	if err := json.Unmarshal(bytes, &receipts); err != nil {
		return nil, err
	}

	return receipts, nil
}

func (bc *Blockchain) SaveTxIndex(txHash common.Hash, blockNum uint64) error {
	path := filepath.Join(bc.dataDir, "txindex.json")

	index := map[string]uint64{}
	data, _ := os.ReadFile(path)
	if len(data) > 0 {
		json.Unmarshal(data, &index)
	}

	index[txHash.Hex()] = blockNum

	out, _ := json.MarshalIndent(index, "", "  ")
	return os.WriteFile(path, out, 0o644)
}

func (bc *Blockchain) FindTxBlock(txHash common.Hash) (uint64, error) {
	path := filepath.Join(bc.dataDir, "txindex.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	index := map[string]uint64{}
	if err := json.Unmarshal(data, &index); err != nil {
		return 0, err
	}

	blockNum, ok := index[txHash.Hex()]
	if !ok {
		return 0, fmt.Errorf("tx not found")
	}

	return blockNum, nil
}
