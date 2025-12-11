// send-gorr.js
const { ethers } = require("ethers");

async function main() {
  // 1. RPC provider naar jouw node
  const provider = new ethers.JsonRpcProvider("http://127.0.0.1:9000", {
    name: "gorrillazz",
    chainId: 9999,          // jouw chainId
  });

  // 2. Admin private key uit data/wallets.json
  //    LET OP: voeg "0x" toe voor de hex key
  const adminPrivateKey = "8685f1623ece272cde76efa71074da44d542e8036c2fe1e535ea6d1ee47987fd"; // <-- aanpassen

  const wallet = new ethers.Wallet(adminPrivateKey, provider);

  // 3. Ontvanger = Treasury-adres
  const to = "0x2F74AF61214E89796C37966d4B674a5aE148aa82"; // treasury

  // 4. Hoeveel GORR wil je sturen?
  //    1 GORR = 10^18 (18 decimals)
  const value = ethers.parseUnits("1.0", 18); // 1 GORR

  // 5. Bouw transactie
  const txRequest = {
    to,
    value,
    gasLimit: 21000n,
    gasPrice: ethers.parseUnits("1", "gwei"), // kan laag, is toch private chain
  };

  console.log("Admin address:", await wallet.getAddress());
  console.log("Sending 1 GORR to:", to);

  // 6. Versturen
  const txResponse = await wallet.sendTransaction(txRequest);
  console.log("Tx sent, hash:", txResponse.hash);

  // 7. Wachten tot het in een block zit
  const receipt = await txResponse.wait();
  console.log("Tx mined in block:", receipt.blockNumber);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
