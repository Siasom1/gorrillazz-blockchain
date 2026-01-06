const { ethers } = require("ethers");

const PRIVATE_KEY = "b0f70de5ae4d1ab5ee7b81323d0f0651be908956db729f8cc380a0c1f239a9e9";

async function main() {
  // ⚠️ Force network config
  const provider = new ethers.providers.JsonRpcProvider(
    "http://localhost:9000",
    {
      name: "gorrillazz",
      chainId: 9999
    }
  );

  // Force provider to lock network
  await provider.getNetwork();

  const wallet = new ethers.Wallet(PRIVATE_KEY, provider);

  const nonce = await provider.getTransactionCount(wallet.address);

  const tx = {
    to: "0x227C2b4C4511CCAdb40A5B7Ee603Ab0D056951c5",
    value: ethers.utils.parseEther("1"),
    gasLimit: 21000,
    gasPrice: 0,
    nonce: nonce,
    chainId: 9999,   // 🔥 CRUCIAAL
    type: 0          // 🔥 legacy tx (geen EIP-1559)
  };

  const signedTx = await wallet.signTransaction(tx);
  console.log("RAW_TX:", signedTx);

  const sent = await provider.sendTransaction(signedTx);
  console.log("TX HASH:", sent.hash);
}

main().catch(console.error);
