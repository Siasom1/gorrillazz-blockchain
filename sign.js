const { ethers } = require("ethers");

// ⚠️ ADMIN private key (MET 0x ervoor)
const pk = "8685f1623ece272cde76efa71074da44d542e8036c2fe1e535ea6d1ee47987fd";

// ⚠️ jouw MetaMask ontvang-adres
const to = "0x227C2b4C4511CCAdb40A5B7Ee603Ab0D056951c5";

(async () => {
  const wallet = new ethers.Wallet(pk);

  const tx = {
    nonce: 0,                    // check met eth_getTransactionCount
    gasPrice: ethers.parseUnits("0", "gwei"),
    gasLimit: 21000,
    to,
    value: ethers.parseEther("1"),
    chainId: 9999,

    // 🔴 DIT Dwingt legacy tx af
    type: 0
  };

  const signed = await wallet.signTransaction(tx);
  console.log(signed);
})();
