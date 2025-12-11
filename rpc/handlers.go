package rpc

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	payment_gateway "github.com/Siasom1/gorrillazz-chain/modules/payment_gateway"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// zelfde fee-config als in de block producer
	treasuryFeeBps = 250   // 2.5%
	bpsDenominator = 10000 // 100%
	// simpele expiry op RPC-laag (voor UI): 15 minuten
	defaultIntentExpirySeconds = 15 * 60
)

type RPCHandlers struct {
	Chain *blockchain.Blockchain
}

// ---------------------------------------------------------------
// REGISTER METHODS
// ---------------------------------------------------------------

func NewHandlers(chain *blockchain.Blockchain) map[string]RPCHandler {
	h := &RPCHandlers{Chain: chain}

	return map[string]RPCHandler{
		// ---------------- GORR RPC ----------------
		"gorr_blockNumber":          h.blockNumber,
		"gorr_getHead":              h.getHead,
		"gorr_getBlockByNumber":     h.getBlockByNumber,
		"gorr_getBalances":          h.gorrGetBalances,
		"gorr_getUSDCcBalance":      h.gorrGetUSDCcBalance,
		"gorr_createPaymentIntent":  h.gorrCreatePaymentIntent,
		"gorr_getPaymentIntent":     h.gorrGetPaymentIntent,
		"gorr_payInvoice":           h.gorrPayInvoice,
		"gorr_refundInvoice":        h.gorrRefundInvoice,
		"gorr_listMerchantPayments": h.gorrListMerchantPayments,

		// --------- ETHEREUM COMPATIBLE RPC ---------
		"eth_chainId":               h.ethChainId,
		"net_version":               h.netVersion,
		"eth_blockNumber":           h.ethBlockNumber,
		"eth_getBlockByNumber":      h.ethGetBlockByNumber,
		"eth_getBalance":            h.ethGetBalance,
		"eth_getTransactionCount":   h.ethGetTransactionCount,
		"eth_sendRawTransaction":    h.ethSendRawTransaction,
		"eth_getTransactionReceipt": h.ethGetTransactionReceipt,
		"eth_getTransactionByHash":  h.ethGetTransactionByHash,
		"eth_gasPrice":              h.ethGasPrice,
		"eth_maxPriorityFeePerGas":  h.ethMaxPriorityFeePerGas,
		"eth_feeHistory":            h.ethFeeHistory,
	}
}

// ============================================================================
// HELPERS
// ============================================================================

// paymentIntentView maakt een frontend-vriendelijk JSON-object
// zonder extra aannames over nieuwe velden in PaymentIntent.
func paymentIntentView(intent *payment_gateway.PaymentIntent) map[string]interface{} {
	if intent == nil {
		return nil
	}

	// fee / net worden client-side óf hier berekend op basis van Amount
	fee := new(big.Int).Mul(intent.Amount, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))

	net := new(big.Int).Sub(intent.Amount, fee)

	expiry := intent.Timestamp + defaultIntentExpirySeconds
	now := uint64(time.Now().Unix())
	expired := now > expiry

	status := "pending"
	if intent.Refunded {
		status = "refunded"
	} else if intent.Paid {
		if expired {
			status = "paid_expired"
		} else {
			status = "paid"
		}
	} else if expired {
		status = "expired"
	}

	return map[string]interface{}{
		"id":          intent.ID,
		"merchant":    intent.Merchant.Hex(),
		"payer":       intent.Payer.Hex(),
		"amount":      intent.Amount.String(), // dec string voor frontend
		"token":       intent.Token,
		"timestamp":   intent.Timestamp,
		"paid":        intent.Paid,
		"refunded":    intent.Refunded,
		"feeBps":      treasuryFeeBps,
		"grossAmount": intent.Amount.String(),
		"feeAmount":   fee.String(),
		"netAmount":   net.String(),
		"expiry":      expiry,
		"expired":     expired,
		"status":      status,
	}
}

// safeAddrString geeft "0x000..." bij lege address
func safeAddrString(addr common.Address) string {
	if addr == (common.Address{}) {
		return "0x0000000000000000000000000000000000000000"
	}
	return addr.Hex()
}

// ============================================================================
// GORR RPC IMPLEMENTATION
// ============================================================================

func (h *RPCHandlers) blockNumber(params []interface{}) (interface{}, error) {
	return h.Chain.Head().Header.Number, nil
}

func (h *RPCHandlers) getHead(params []interface{}) (interface{}, error) {
	return h.Chain.Head(), nil
}

func (h *RPCHandlers) getBlockByNumber(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing block number")
	}

	numFloat, ok := params[0].(float64)
	if !ok {
		return nil, errors.New("block number must be float")
	}

	n := uint64(numFloat)
	return h.Chain.LoadBlock(n)
}

// ============================================================================
// PAYMENT GATEWAY RPC
// ============================================================================

// gorr_createPaymentIntent(merchant, amountDec, token) → { id, intentView }
func (h *RPCHandlers) gorrCreatePaymentIntent(params []interface{}) (interface{}, error) {
	if len(params) < 3 {
		return nil, errors.New("missing params: merchant, amount, token")
	}

	merchantStr, ok := params[0].(string)
	if !ok {
		return nil, errors.New("merchant must be string")
	}
	amountStr, ok := params[1].(string)
	if !ok {
		return nil, errors.New("amount must be string (decimal)")
	}
	token, ok := params[2].(string)
	if !ok {
		return nil, errors.New("token must be string")
	}

	merchant := common.HexToAddress(merchantStr)
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return nil, errors.New("invalid decimal amount")
	}

	intent, id, err := h.Chain.Payment.CreateIntent(
		merchant,
		amount,
		token,
		uint64(time.Now().Unix()),
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":     id,
		"intent": paymentIntentView(intent),
	}, nil
}

// gorr_getPaymentIntent(id) → intentView
func (h *RPCHandlers) gorrGetPaymentIntent(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing payment intent id")
	}

	// JSON numbers komen als float64 binnen
	idFloat, ok := params[0].(float64)
	if !ok {
		return nil, errors.New("id must be number")
	}
	id := uint64(idFloat)

	intent, err := h.Chain.Payment.GetIntent(id)
	if err != nil {
		return nil, err
	}

	return paymentIntentView(intent), nil
}

// gorr_payInvoice(id, payerAddr) → intentView
func (h *RPCHandlers) gorrPayInvoice(params []interface{}) (interface{}, error) {
	if len(params) < 2 {
		return nil, errors.New("missing params: id, payer")
	}

	idFloat, ok := params[0].(float64)
	if !ok {
		return nil, errors.New("id must be number")
	}
	id := uint64(idFloat)

	payerStr, ok := params[1].(string)
	if !ok {
		return nil, errors.New("payer must be string")
	}
	payer := common.HexToAddress(payerStr)

	intent, err := h.Chain.Payment.PayIntent(id, payer)
	if err != nil {
		return nil, err
	}

	return paymentIntentView(intent), nil
}

// gorr_refundInvoice(id) → intentView
func (h *RPCHandlers) gorrRefundInvoice(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing id param")
	}

	idFloat, ok := params[0].(float64)
	if !ok {
		return nil, errors.New("id must be number")
	}
	id := uint64(idFloat)

	intent, err := h.Chain.Payment.RefundIntent(id)
	if err != nil {
		return nil, err
	}

	return paymentIntentView(intent), nil
}

// gorr_listMerchantPayments(merchantAddr) → []intentView
func (h *RPCHandlers) gorrListMerchantPayments(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing merchant address")
	}

	addrStr, ok := params[0].(string)
	if !ok {
		return nil, errors.New("merchant must be string")
	}
	merchant := common.HexToAddress(addrStr)

	intents := h.Chain.Payment.ListMerchantPayments(merchant)
	result := make([]map[string]interface{}, 0, len(intents))
	for _, in := range intents {
		result = append(result, paymentIntentView(in))
	}
	return result, nil
}

// ============================================================================
// ETHEREUM COMPATIBLE RPC IMPLEMENTATION
// ============================================================================

func (h *RPCHandlers) ethChainId(params []interface{}) (interface{}, error) {
	return fmt.Sprintf("0x%x", h.Chain.NetworkID()), nil
}

func (h *RPCHandlers) netVersion(params []interface{}) (interface{}, error) {
	return fmt.Sprintf("%d", h.Chain.NetworkID()), nil
}

func (h *RPCHandlers) ethBlockNumber(params []interface{}) (interface{}, error) {
	num := h.Chain.Head().Header.Number
	return fmt.Sprintf("0x%x", num), nil
}

func (h *RPCHandlers) ethGetBlockByNumber(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing block number")
	}

	numHex, ok := params[0].(string)
	if !ok {
		return nil, errors.New("block number must be hex string")
	}

	var blockNum uint64
	if numHex == "latest" {
		blockNum = h.Chain.Head().Header.Number
	} else {
		if len(numHex) < 3 || !strings.HasPrefix(numHex, "0x") {
			return nil, errors.New("invalid hex block number")
		}
		n, err := strconv.ParseUint(numHex[2:], 16, 64)
		if err != nil {
			return nil, err
		}
		blockNum = n
	}

	block, err := h.Chain.LoadBlock(blockNum)
	if err != nil {
		return nil, err
	}

	resp := map[string]interface{}{
		"number":           fmt.Sprintf("0x%x", block.Header.Number),
		"hash":             block.Hash().Hex(),
		"parentHash":       block.Header.ParentHash.Hex(),
		"nonce":            "0x0000000000000000",
		"sha3Uncles":       "0x1dcc4de8dec75d7aab85b567b6ccb1ec3290a3d88f3a12b3e76c1f60e3e3a39a",
		"logsBloom":        "0x" + strings.Repeat("0", 512),
		"transactionsRoot": block.Header.TxRoot.Hex(),
		"stateRoot":        block.Header.StateRoot.Hex(),
		"receiptsRoot":     "0x" + strings.Repeat("0", 64),
		"miner":            "0x0000000000000000000000000000000000000000",
		"difficulty":       "0x0",
		"totalDifficulty":  "0x0",
		"extraData":        "0x",
		"gasLimit":         "0x1c9c380",
		"gasUsed":          "0x0",
		"timestamp":        fmt.Sprintf("0x%x", block.Header.Time),
		"uncles":           []string{},
	}

	// Ethereum verwacht een array (niet null)
	txs := []interface{}{}
	for _, tx := range block.Transactions {
		txs = append(txs, tx.Hash().Hex())
	}
	resp["transactions"] = txs

	return resp, nil
}

func (h *RPCHandlers) ethGetTransactionReceipt(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing tx hash")
	}

	hashStr, ok := params[0].(string)
	if !ok {
		return nil, errors.New("tx hash must be string")
	}
	txHash := common.HexToHash(hashStr)

	blockNum, err := h.Chain.FindTxBlock(txHash)
	if err != nil {
		return nil, err
	}

	receipts, err := h.Chain.LoadReceipts(blockNum)
	if err != nil {
		return nil, err
	}

	for _, r := range receipts {
		if r.TxHash == txHash {
			// types.Receipt wordt direct als JSON teruggegeven
			return r, nil
		}
	}

	return nil, errors.New("receipt not found")
}

func (h *RPCHandlers) ethGetTransactionByHash(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing tx hash")
	}

	hashStr, ok := params[0].(string)
	if !ok {
		return nil, errors.New("tx hash must be string")
	}
	txHash := common.HexToHash(hashStr)

	blockNum, err := h.Chain.FindTxBlock(txHash)
	if err != nil {
		return nil, err
	}

	block, err := h.Chain.LoadBlock(blockNum)
	if err != nil {
		return nil, err
	}

	for idx, tx := range block.Transactions {
		if tx.Hash() == txHash {
			from := tx.Sender
			if from == (common.Address{}) {
				// Fallback voor oude txs (DEV)
				if h.Chain.AdminAddr != (common.Address{}) {
					from = h.Chain.AdminAddr
				} else {
					from = common.HexToAddress("0x936808d3950Dab542bEF8E71D2d7d36A0bB538ec")
				}
			}

			var toStr string
			if tx.To == nil {
				toStr = "0x0000000000000000000000000000000000000000"
			} else {
				toStr = tx.To.Hex()
			}

			result := map[string]interface{}{
				"hash":             tx.Hash().Hex(),
				"nonce":            fmt.Sprintf("0x%x", tx.Nonce),
				"blockNumber":      fmt.Sprintf("0x%x", blockNum),
				"transactionIndex": fmt.Sprintf("0x%x", idx),
				"from":             from.Hex(),
				"to":               toStr,
				"value":            fmt.Sprintf("0x%x", tx.Value),
				"gas":              fmt.Sprintf("0x%x", tx.Gas),
				"gasPrice":         fmt.Sprintf("0x%x", tx.GasPrice),
				"input":            "0x" + common.Bytes2Hex(tx.Data),
			}
			return result, nil
		}
	}

	return nil, errors.New("tx not found")
}

// ------------------------------
// BALANCE
// ------------------------------

func (h *RPCHandlers) ethGetBalance(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing address")
	}

	addrStr, ok := params[0].(string)
	if !ok {
		return nil, errors.New("address must be string")
	}
	addr := common.HexToAddress(addrStr)

	bal, err := h.Chain.State.GetBalance(addr)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("0x%x", bal), nil
}

// ------------------------------
// NONCE
// ------------------------------

func (h *RPCHandlers) ethGetTransactionCount(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing address")
	}

	addrStr, ok := params[0].(string)
	if !ok {
		return nil, errors.New("address must be string")
	}
	addr := common.HexToAddress(addrStr)

	nonce, err := h.Chain.State.GetNonce(addr)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("0x%x", nonce), nil
}

// ---------------------------------------------------------------
// eth_sendRawTransaction — Accept signed RLP encoded tx
// ---------------------------------------------------------------

func (h *RPCHandlers) ethSendRawTransaction(params []interface{}) (interface{}, error) {
	fmt.Println("[RPC] eth_sendRawTransaction CALLED with params:", params)

	if len(params) < 1 {
		return nil, errors.New("missing raw transaction data")
	}

	rawHex, ok := params[0].(string)
	if !ok {
		return nil, errors.New("raw transaction must be a hex string")
	}

	rawBytes, err := hexutil.Decode(rawHex)
	if err != nil {
		return nil, err
	}

	tx, err := types.DecodeTx(rawBytes)
	if err != nil {
		return nil, err
	}

	// DEV: zet een betrouwbare sender (admin) als fallback
	sender := h.Chain.AdminAddr
	if sender == (common.Address{}) {
		sender = common.HexToAddress("0x936808d3950Dab542bEF8E71D2d7d36A0bB538ec")
	}
	tx.Sender = sender

	if err := h.Chain.TxPool.Add(tx); err != nil {
		return nil, err
	}

	return tx.Hash().Hex(), nil
}

// ---------------------------------------------------------------
// gorr_getUSDCcBalance(address)
// ---------------------------------------------------------------

func (h *RPCHandlers) gorrGetUSDCcBalance(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing address")
	}

	addrHex, ok := params[0].(string)
	if !ok {
		return nil, errors.New("address must be string")
	}

	addr := common.HexToAddress(addrHex)

	bal, err := h.Chain.State.GetUSDCcBalance(addr)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("0x%x", bal), nil
}

// ---------------------------------------------------------------
// gorr_getBalances(address)  → { "GORR": "0x..", "USDCc": "0x.." }
// ---------------------------------------------------------------

func (h *RPCHandlers) gorrGetBalances(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing address")
	}

	addrHex, ok := params[0].(string)
	if !ok {
		return nil, errors.New("address must be string")
	}

	addr := common.HexToAddress(addrHex)

	gorrBal, err := h.Chain.State.GetBalance(addr)
	if err != nil {
		return nil, err
	}

	usdcBal, err := h.Chain.State.GetUSDCcBalance(addr)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"GORR":  fmt.Sprintf("0x%x", gorrBal),
		"USDCc": fmt.Sprintf("0x%x", usdcBal),
	}, nil
}

// ---------------------------------------------------------------
// Gas price helpers
// ---------------------------------------------------------------

func (h *RPCHandlers) ethGasPrice(params []interface{}) (interface{}, error) {
	// Statistische 1 gwei
	return "0x3b9aca00", nil // 1_000_000_000 wei
}

func (h *RPCHandlers) ethMaxPriorityFeePerGas(params []interface{}) (interface{}, error) {
	// Private chain → constant tip
	return "0x3b9aca00", nil
}

func (h *RPCHandlers) ethFeeHistory(params []interface{}) (interface{}, error) {
	result := map[string]interface{}{
		"oldestBlock": "0x0",
		"reward":      [][]string{{"0x3b9aca00"}}, // 1 gwei tip
		"baseFeePerGas": []string{
			"0x3b9aca00", // 1 gwei base fee
		},
		"gasUsedRatio": []float64{0.5},
	}
	return result, nil
}
