require("hardhat-gas-reporter");
require("@nomiclabs/hardhat-ethers");
require("@nomicfoundation/hardhat-verify");
require("@nomiclabs/hardhat-waffle");
require("dotenv").config();
const hre = require("hardhat");

// npx hardhat run scripts/UpgradeSepolia.js --network sepolia

// Fill in these constants before running
const TELLOR_MASTER = "0x80fc34a2f9FfE86F41580F47368289C402DEc660"; // proxy (token address)
const TELLOR_360_CONTRACT = "0x726737F28EA0BA5D23e16d1C3bb852982ff8651A"; // current implementation
const TELLOR_TOKEN_BRIDGE_NEW = "0x0000000000000000000000000000000000000000"; // deployed token bridge new
const UPDATE_ORACLE_TESTNET_IMPL = "0x0000000000000000000000000000000000000000"; // deployed UpdateOracleTestnet(address newTokenBridge)

async function main(pk, rpcUrl) {
  await hre.run("compile");

  const provider = new hre.ethers.providers.JsonRpcProvider(rpcUrl);
  const wallet = new hre.ethers.Wallet(pk, provider);

  const masterAdmin = await hre.ethers.getContractAt(
    "contracts/tellor360/oldContracts/contracts/tellor3/TellorMaster.sol:TellorMaster",
    TELLOR_MASTER,
    wallet
  );

  const tellorAsITellor = await hre.ethers.getContractAt(
    "contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor",
    TELLOR_MASTER,
    wallet
  );

  const updateOracleViaProxy = await hre.ethers.getContractAt(
    "contracts/testing/UpdateOracleTestnet.sol:UpdateOracleTestnet",
    TELLOR_MASTER,
    wallet
  );

  const HASH_ORACLE_CONTRACT = hre.ethers.utils.solidityKeccak256(["string"], ["_ORACLE_CONTRACT"]);
  const HASH_TELLOR_CONTRACT = hre.ethers.utils.solidityKeccak256(["string"], ["_TELLOR_CONTRACT"]);

  console.log("\nTellor Sepolia Upgrade - Oracle redirect via UpdateOracleTestnet");
  console.log("Network:", hre.network.name);
  console.log("Wallet:", wallet.address);
  console.log("TellorMaster (proxy):", TELLOR_MASTER);
  console.log("Tellor360 (impl):", TELLOR_360_CONTRACT);
  console.log("UpdateOracleTestnet (impl):", UPDATE_ORACLE_TESTNET_IMPL);

  const beforeImpl = await tellorAsITellor.getAddressVars(HASH_TELLOR_CONTRACT);
  const beforeOracle = await tellorAsITellor.getAddressVars(HASH_ORACLE_CONTRACT);
  console.log("Before - _TELLOR_CONTRACT:", beforeImpl);
  console.log("Before - _ORACLE_CONTRACT:", beforeOracle);

  // 1) Point implementation to UpdateOracleTestnet
  console.log("\n1) changeTellorContract -> UpdateOracleTestnet...");
  let tx = await masterAdmin.changeTellorContract(UPDATE_ORACLE_TESTNET_IMPL);
  await tx.wait();
  console.log("   tx:", tx.hash);
  _newTellorContract = await tellorAsITellor.getAddressVars(HASH_TELLOR_CONTRACT);
  console.log("   newTellorContract:", _newTellorContract);
  if (_newTellorContract.toLowerCase() !== UPDATE_ORACLE_TESTNET_IMPL.toLowerCase()) {
    throw new Error("Post-check failed: _TELLOR_CONTRACT not updated to UpdateOracleTestnet");
  }

  // 2) Call init() via proxy (writes _ORACLE_CONTRACT to TellorMaster storage)
  console.log("2) Calling init() on proxy (UpdateOracleTestnet)...");
  tx = await updateOracleViaProxy.init();
  await tx.wait();
  console.log("   tx:", tx.hash);
  _newOracleContract = await tellorAsITellor.getAddressVars(HASH_ORACLE_CONTRACT);
  console.log("   newOracleContract:", _newOracleContract);
  if (_newOracleContract.toLowerCase() !== TELLOR_TOKEN_BRIDGE_NEW.toLowerCase()) {
    throw new Error("Post-check failed: _ORACLE_CONTRACT not updated to UpdateOracleTestnet");
  }

  // 3) Point implementation back to Tellor360
  console.log("3) changeTellorContract -> Tellor360...");
  tx = await masterAdmin.changeTellorContract(TELLOR_360_CONTRACT);
  await tx.wait();
  console.log("   tx:", tx.hash);

  const afterImpl = await tellorAsITellor.getAddressVars(HASH_TELLOR_CONTRACT);
  const afterOracle = await tellorAsITellor.getAddressVars(HASH_ORACLE_CONTRACT);
  console.log("\nAfter - _TELLOR_CONTRACT:", afterImpl);
  console.log("After - _ORACLE_CONTRACT:", afterOracle);

  if (afterImpl.toLowerCase() !== TELLOR_360_CONTRACT.toLowerCase()) {
    throw new Error("Post-check failed: _TELLOR_CONTRACT not restored to Tellor360");
  }
  console.log("\nSuccess: Implementation restored and _ORACLE_CONTRACT updated.");
}

main(process.env.TESTNET_PK, process.env.NODE_URL_SEPOLIA_TESTNET)
  .then(() => process.exit(0))
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });


