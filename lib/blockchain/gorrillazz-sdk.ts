// lib/blockchain/gorrillazz-sdk.ts
import {
  JsonRpcProvider,
  Signer,
  TransactionResponse,
  BigNumberish,
  toUtf8Bytes,
} from "ethers";

// -----------------------------------------------------------------------------
// CONFIG
// -----------------------------------------------------------------------------

/**
 * RPC endpoint van je gorrillazzd node.
 * - In dev:  http://localhost:9000
 * - In prod: via environment variable, bijv:
 *   NEXT_PUBLIC_GORR_RPC_URL="https://rpc.gorrillazz.yourdomain.com"
 */
const RPC_URL =
  process.env.GORR_RPC_URL ||
  process.env.NEXT_PUBLIC_GORR_RPC_URL ||
  "http://localhost:9000";

let _provider: JsonRpcProvider | null = null;

export function getGorrProvider(): JsonRpcProvider {
  if (!_provider) {
    _provider = new JsonRpcProvider(RPC_URL);
  }
  return _provider;
}

// -----------------------------------------------------------------------------
// TYPES
// -----------------------------------------------------------------------------

export interface GorrBalances {
  GORR: string; // hex string in wei
  USDCc: string; // hex string in wei
}

// Matcht de JSON-shape uit rpc/handlers.go → paymentIntentView
export interface PaymentIntentView {
  id: number;
  merchant: string;
  payer: string;
  amount: string; // decimal string (wei)
  token: string;
  timestamp: number;
  paid: boolean;
  refunded: boolean;
  feeBps: number;
  grossAmount: string;
  feeAmount: string;
  netAmount: string;
  expiry: number;
  expired: boolean;
  status: string; // "pending" | "paid" | "expired" | "paid_expired" | "refunded"
}

interface JsonRpcRequest {
  jsonrpc: "2.0";
  method: string;
  params: any[];
  id: number;
}

interface JsonRpcSuccess<T> {
  jsonrpc: "2.0";
  result: T;
  id: number;
}

interface JsonRpcError {
  jsonrpc: "2.0";
  error: {
    code?: number;
    message: string;
    data?: any;
  };
  id: number;
}

// -----------------------------------------------------------------------------
// Low-level JSON-RPC helper
// -----------------------------------------------------------------------------

let _rpcId = 1;

async function jsonRpc<T = any>(method: string, params: any[] = []): Promise<T> {
  const body: JsonRpcRequest = {
    jsonrpc: "2.0",
    method,
    params,
    id: _rpcId++,
  };

  const res = await fetch(RPC_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    // Next.js: zowel op server als client bruikbaar
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `RPC HTTP error (${res.status}): ${res.statusText} ${text || ""}`
    );
  }

  const json = (await res.json()) as JsonRpcSuccess<T> | JsonRpcError;

  if ("error" in json && json.error) {
    throw new Error(
      `RPC ${method} error: ${json.error.message || "unknown error"}`
    );
  }

  return (json as JsonRpcSuccess<T>).result;
}

// -----------------------------------------------------------------------------
// PUBLIC READ HELPERS
// -----------------------------------------------------------------------------

/**
 * Haal chainId (hex string) op van je node.
 */
export async function getChainIdHex(): Promise<string> {
  return jsonRpc<string>("eth_chainId", []);
}

/**
 * Haal netVersion op als string (bv. "9999").
 */
export async function getNetVersion(): Promise<string> {
  return jsonRpc<string>("net_version", []);
}

/**
 * Haal GORR & USDCc balances op voor een adres.
 * Retourneert hex strings (wei).
 */
export async function getGorrBalances(address: string): Promise<GorrBalances> {
  return jsonRpc<GorrBalances>("gorr_getBalances", [address]);
}

/**
 * Standaard eth_getBalance helper voor ruwe GORR balance.
 * Retourneert hex string (wei).
 */
export async function getEthBalanceHex(address: string): Promise<string> {
  return jsonRpc<string>("eth_getBalance", [address, "latest"]);
}

/**
 * Haal een payment intent op op basis van id.
 */
export async function getPaymentIntent(id: number): Promise<PaymentIntentView> {
  return jsonRpc<PaymentIntentView>("gorr_getPaymentIntent", [id]);
}

/**
 * Haal alle payment intents voor een merchant op.
 */
export async function listMerchantPayments(
  merchantAddress: string
): Promise<PaymentIntentView[]> {
  return jsonRpc<PaymentIntentView[]>("gorr_listMerchantPayments", [
    merchantAddress,
  ]);
}

// -----------------------------------------------------------------------------
// PAYMENT GATEWAY HELPERS
// -----------------------------------------------------------------------------

/**
 * Maak een payment intent aan:
 * - merchant: merchant address (0x…)
 * - amountWei: BigNumberish in wei (bv. ethers.parseEther("1"))
 * - token: "GORR" of "USDCc" (nu vooral "GORR")
 *
 * Retourneert: { id, intent }
 */
export async function createPaymentIntent(
  merchant: string,
  amountWei: BigNumberish,
  token: string = "GORR"
): Promise<{ id: number; intent: PaymentIntentView }> {
  // RPC verwacht decimal string, niet hex
  const amount =
    typeof amountWei === "bigint"
      ? amountWei.toString()
      : typeof amountWei === "string"
      ? amountWei
      : (amountWei as any).toString();

  return jsonRpc<{ id: number; intent: PaymentIntentView }>(
    "gorr_createPaymentIntent",
    [merchant, amount, token]
  );
}

/**
 * Markeer een intent als betaald (alleen PaymentGateway status),
 * NIET de daadwerkelijke on-chain transfer (die gaat via sendGorrPaymentTx).
 */
export async function markInvoicePaid(
  id: number,
  payerAddress: string
): Promise<PaymentIntentView> {
  return jsonRpc<PaymentIntentView>("gorr_payInvoice", [id, payerAddress]);
}

/**
 * Vraag refund aan via PaymentGateway.
 * (on-chain logic: markeert intent als refunded, etc.)
 */
export async function refundInvoice(
  id: number
): Promise<PaymentIntentView> {
  return jsonRpc<PaymentIntentView>("gorr_refundInvoice", [id]);
}

// -----------------------------------------------------------------------------
// ON-CHAIN BETALING: GORR PAYMENT TX
// -----------------------------------------------------------------------------

export interface SendGorrPaymentTxParams {
  /**
   * Merchant address uit de payment intent.
   */
  to: string;
  /**
   * Totaal bedrag (gross) in wei.
   * De chain splitst fee / net op basis van treasuryFeeBps.
   */
  valueWei: BigNumberish;
  /**
   * Optioneel: payment intent id.
   * Als gezet → tx.data = "GORR_PAY:<id>" (ASCII).
   */
  intentId?: number;
  /**
   * Optioneel: custom gasLimit (standard is meestal genoeg).
   */
  gasLimit?: BigNumberish;
}

/**
 * Verstuur een GORR payment transaction via een ethers.js Signer.
 *
 * - Als intentId gezet is:
 *   - tx.data = "GORR_PAY:<id>" (ASCII → bytes)
 *   - block producer herkent dit als payment tx
 *   - PaymentGateway.MarkPaidFromTx wordt aangeroepen + fee/tax splits
 *
 * Signer kan zijn:
 * - een wallet met provider
 * - een JsonRpcSigner van MetaMask in de browser
 */
export async function sendGorrPaymentTx(
  signer: Signer,
  params: SendGorrPaymentTxParams
): Promise<TransactionResponse> {
  const { to, valueWei, intentId, gasLimit } = params;

  // Zorg dat signer een provider heeft (nodig voor gas price / chain id, etc.)
  if (!signer.provider) {
    signer = signer.connect(getGorrProvider());
  }

  const tx: any = {
    to,
    value: valueWei,
  };

  if (intentId !== undefined) {
    const dataStr = `GORR_PAY:${intentId}`;
    tx.data = toUtf8Bytes(dataStr);
  }

  if (gasLimit !== undefined) {
    tx.gasLimit = gasLimit;
  }

  // Laat ethers zelf gasPrice en nonce resolven via provider
  const response = await signer.sendTransaction(tx);
  return response;
}

// -----------------------------------------------------------------------------
// TRANSACTION LOOKUPS
// -----------------------------------------------------------------------------

/**
 * eth_getTransactionByHash wrapper.
 * Retourneert ruwe tx map zoals node terugstuurt.
 */
export async function getTransactionByHash(
  txHash: string
): Promise<Record<string, any>> {
  return jsonRpc<Record<string, any>>("eth_getTransactionByHash", [txHash]);
}

/**
 * eth_getTransactionReceipt wrapper.
 */
export async function getTransactionReceipt(
  txHash: string
): Promise<Record<string, any> | null> {
  try {
    const res = await jsonRpc<Record<string, any>>(
      "eth_getTransactionReceipt",
      [txHash]
    );
    return res;
  } catch (err: any) {
    // Je node geeft "tx not indexed" als hij 'm niet kent.
    if (err?.message?.includes("tx not indexed")) {
      return null;
    }
    throw err;
  }
}
// ============================================================================
// PRODUCTION NETWORK CONFIG (MAINNET)
// Added helpers – NO breaking changes
// ============================================================================

import { BrowserProvider, ethers } from "ethers";

/*
 MAINNET
*/
export const GORR_CHAIN_ID = 799797;
export const GORR_CHAIN_ID_HEX = "0xC34F5";

export const GORR_NETWORK_PARAMS = {
  chainId: GORR_CHAIN_ID_HEX,
  chainName: "gorrillazz",
  rpcUrls: [RPC_URL],
  nativeCurrency: {
    name: "Gorrillazz",
    symbol: "GORR",
    decimals: 18,
  },
  blockExplorerUrls: [RPC_URL],
};

// ============================================================================
// WALLET (MetaMask / Trust / etc)
// ============================================================================

export function getInjectedProvider(): BrowserProvider | null {
  if (typeof window === "undefined") return null;
  if (!(window as any).ethereum) return null;

  return new BrowserProvider((window as any).ethereum);
}

export async function connectWalletSigner(): Promise<Signer> {
  const provider = getInjectedProvider();
  if (!provider) throw new Error("Wallet not found");

  await provider.send("eth_requestAccounts", []);
  return provider.getSigner();
}

// ============================================================================
// AUTO NETWORK SWITCH / ADD
// ============================================================================

export async function ensureGorrNetwork(): Promise<void> {
  if (typeof window === "undefined") return;

  const eth = (window as any).ethereum;
  if (!eth) return;

  const current = await eth.request({ method: "eth_chainId" });

  if (current === GORR_CHAIN_ID_HEX) return;

  try {
    await eth.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: GORR_CHAIN_ID_HEX }],
    });
  } catch {
    await eth.request({
      method: "wallet_addEthereumChain",
      params: [GORR_NETWORK_PARAMS],
    });
  }
}

// ============================================================================
// SIMPLE SEND (no intent)
// ============================================================================

export async function sendGORR(
  to: string,
  amountEther: string
): Promise<string> {
  const signer = await connectWalletSigner();
  await ensureGorrNetwork();

  const tx = await signer.sendTransaction({
    to,
    value: ethers.parseEther(amountEther),
  });

  return tx.hash;
}

// ============================================================================
// REALTIME HELPERS
// ============================================================================

/*
 Listen for new blocks (merchant dashboard live updates)
*/
export function onNewBlock(cb: (blockNumber: number) => void) {
  const provider = getGorrProvider();
  provider.on("block", cb);

  return () => provider.off("block", cb);
}

/*
 Wait for tx confirmation
*/
export async function waitForTx(txHash: string) {
  const provider = getGorrProvider();
  return provider.waitForTransaction(txHash);
}

// ============================================================================
// ADMIN HELPERS (optional – for dashboard)
// ============================================================================

export async function adminMint(
  to: string,
  amountWei: string,
  token: "GORR" | "USDCc"
) {
  return jsonRpc("admin_mint", [to, amountWei, token]);
}

export async function adminBurn(
  from: string,
  amountWei: string,
  token: "GORR" | "USDCc"
) {
  return jsonRpc("admin_burn", [from, amountWei, token]);
}

