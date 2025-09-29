require("@nomiclabs/hardhat-ethers");
require("dotenv").config();
const hre = require("hardhat");
const { ethers, network } = hre;
const h = require("../test/helpers/evmHelpers");

// REAL: npx hardhat run scripts/UpgradeSepolia.js --network sepolia
// SIMULATED: npx hardhat run scripts/UpgradeSepolia.js --network hardhat

// Real runs depend on NODE_URL_SEPOLIA_TESTNET and TESTNET_PK in .env file
// Simulated runs depend on NODE_URL_SEPOLIA_TESTNET and Sepolia network forking configured in hardhat.config.js

// Fill in these constants before running
const TELLOR_MASTER = "0x80fc34a2f9FfE86F41580F47368289C402DEc660"; // proxy (token address)
const TELLOR_360_CONTRACT = "0x726737F28EA0BA5D23e16d1C3bb852982ff8651A"; // current implementation
const TELLOR_TOKEN_BRIDGE_NEW = "0x87e025f9c3E20E8Cd1a5D2854237D75A4624F72e"; // deployed token bridge new
const UPDATE_ORACLE_TESTNET_IMPL = "0x928707dd5341EAe39cc21ac70161b3DE6f24839e"; // deployed UpdateOracleTestnet(address newTokenBridge)
const TELLOR_TOKEN_BRIDGE_OLD = "0x5acb5977f35b1A91C4fE0F4386eB669E046776F2"; // just for sanity checks
const DEITY = "0x20bEC8F31dea6C13A016DC7fCBdF74f61DC8Ec2c";

async function getSigner(pk, rpcUrl) {
  let signer = null;
  const network_name = hre.network.name;
  if (network_name == "hardhat") {
    console.log("Hardhat network detected: ", network_name);
    await h.impersonateAccount(DEITY);
    signer = await ethers.getSigner(DEITY);
  } else {
    console.log("Non-Hardhat network detected: ", network_name);
    const provider = new hre.ethers.providers.JsonRpcProvider(rpcUrl);
    signer = new hre.ethers.Wallet(pk, provider);
  }
  console.log("Acting as:", await signer.getAddress());

  if (signer.address.toLowerCase() !== DEITY.toLowerCase()) {
    throw new Error("Signer address does not match DEITY address");
  }
 
  return signer;
}

async function main(pk, rpcUrl) {
  await hre.run("compile");
  const signer = await getSigner(pk, rpcUrl);

  const masterDEITY = await hre.ethers.getContractAt(
    "contracts/tellor360/oldContracts/contracts/tellor3/TellorMaster.sol:TellorMaster",
    TELLOR_MASTER,
    signer
  );

  const tellorAsITellor = await hre.ethers.getContractAt(
    "contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor",
    TELLOR_MASTER,
    signer
  );

  const updateOracleViaProxy = await hre.ethers.getContractAt(
    "contracts/testing/UpdateOracleTestnet.sol:UpdateOracleTestnet",
    TELLOR_MASTER,
    signer
  );

  const HASH_ORACLE_CONTRACT = hre.ethers.utils.solidityKeccak256(["string"], ["_ORACLE_CONTRACT"]);
  const HASH_TELLOR_CONTRACT = hre.ethers.utils.solidityKeccak256(["string"], ["_TELLOR_CONTRACT"]);
  console.log("HASH_ORACLE_CONTRACT:", HASH_ORACLE_CONTRACT);
  console.log("HASH_TELLOR_CONTRACT:", HASH_TELLOR_CONTRACT);
  console.log("\nTellor Sepolia Upgrade - Oracle redirect via UpdateOracleTestnet");
  console.log("Network:", hre.network.name);
  console.log("Wallet:", signer.address);
  console.log("TellorMaster (proxy):", TELLOR_MASTER);
  console.log("Tellor360 (impl):", TELLOR_360_CONTRACT);
  console.log("UpdateOracleTestnet (impl):", UPDATE_ORACLE_TESTNET_IMPL);

  const beforeImpl = await tellorAsITellor.callStatic.addresses(HASH_TELLOR_CONTRACT);
  const beforeOracle = await tellorAsITellor.callStatic.addresses(HASH_ORACLE_CONTRACT);
  console.log("Before - _TELLOR_CONTRACT:", beforeImpl);
  console.log("Before - _ORACLE_CONTRACT:", beforeOracle);

  // sanity checks
  console.log("\nSanity checks...");
  if (beforeImpl.toLowerCase() !== TELLOR_360_CONTRACT.toLowerCase()) {
    throw new Error("Pre-check failed: _TELLOR_CONTRACT not pointing to Tellor360");
  }
  if (beforeOracle.toLowerCase() !== TELLOR_TOKEN_BRIDGE_OLD.toLowerCase()) {
    throw new Error("Pre-check failed: _ORACLE_CONTRACT not pointing to TellorTokenBridge");
  }

  // 1) Point implementation to UpdateOracleTestnet
  console.log("\n1) changeTellorContract -> UpdateOracleTestnet...");
  let tx = await masterDEITY.changeTellorContract(UPDATE_ORACLE_TESTNET_IMPL);
  await tx.wait();
  console.log("   tx:", tx.hash);
  _newTellorContract = await tellorAsITellor.callStatic.addresses(HASH_TELLOR_CONTRACT);
  console.log("   newTellorContract:", _newTellorContract);
  if (_newTellorContract.toLowerCase() !== UPDATE_ORACLE_TESTNET_IMPL.toLowerCase()) {
    throw new Error("Post-check failed: _TELLOR_CONTRACT not updated to UpdateOracleTestnet");
  }

  // 2) Call init() via proxy (writes _ORACLE_CONTRACT to TellorMaster storage)
  console.log("2) Calling init() on proxy (UpdateOracleTestnet)...");
  tx = await updateOracleViaProxy.init();
  await tx.wait();
  console.log("   tx:", tx.hash);
  _newOracleContract = await tellorAsITellor.callStatic.addresses(HASH_ORACLE_CONTRACT);
  console.log("   newOracleContract:", _newOracleContract);
  if (_newOracleContract.toLowerCase() !== TELLOR_TOKEN_BRIDGE_NEW.toLowerCase()) {
    throw new Error("Post-check failed: _ORACLE_CONTRACT not updated to UpdateOracleTestnet");
  }

  // 3) Point implementation back to Tellor360
  console.log("3) changeTellorContract -> Tellor360...");
  tx = await masterDEITY.changeTellorContract(TELLOR_360_CONTRACT);
  await tx.wait();
  console.log("   tx:", tx.hash);

  const afterImpl = await tellorAsITellor.callStatic.addresses(HASH_TELLOR_CONTRACT);
  const afterOracle = await tellorAsITellor.callStatic.addresses(HASH_ORACLE_CONTRACT);
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