const { expect } = require("chai");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const { ethers } = require("hardhat");

describe.skip("Testnet Oracle Address Change - Forking Tests", function() {
  // Sepolia addresses
  const TELLOR_MASTER = "0x80fc34a2f9FfE86F41580F47368289C402DEc660"
  const DEV_WALLET = "0x34Fae97547E990ef0E05e05286c51E4645bf1A85"
  const BIGWALLET = "0x20bEC8F31dea6C13A016DC7fCBdF74f61DC8Ec2c"
  const TELLORFLEX = "0xB19584Be015c04cf6CFBF6370Fe94a58b7A38830"
  const OLD_TOKEN_BRIDGE = "0x5acb5977f35b1A91C4fE0F4386eB669E046776F2"
  const UNBONDING_PERIOD = 86400 * 7 * 3 // 3 weeks layer unbonding period
  const DEITY_ADDRESS = "0x20bEC8F31dea6C13A016DC7fCBdF74f61DC8Ec2c"
  const TELLOR_360_CONTRACT = "0x726737F28EA0BA5D23e16d1C3bb852982ff8651A" // current TELLOR_CONTRACT

  const abiCoder = new ethers.utils.AbiCoder()
  let valset_domain_sep_args = abiCoder.encode(["string", "string"], ["checkpoint", "layertest-4"])
  const VALIDATOR_SET_DOMAIN_SEPARATOR_SEPOLIA = h.hash(valset_domain_sep_args)

  let accounts = null
  let flex = null
  let tellor = null
  let devWallet = null
  let deityWallet = null
  let tbridge = null
  let blobstream = null
  let snapshot = null

  before(async function() {
    snapshot = await h.takeSnapshot()
  })

  beforeEach("deploy and setup TokenBridge", async function() {
    await snapshot.restore()
    await h.impersonateAccount(BIGWALLET)
    await h.impersonateAccount(DEV_WALLET)
    await h.impersonateAccount(DEITY_ADDRESS)

    // Get accounts
    accounts = await ethers.getSigners()
    devWallet = await ethers.provider.getSigner(DEV_WALLET)
    bigWallet = await ethers.provider.getSigner(BIGWALLET)
    deityWallet = await ethers.provider.getSigner(DEITY_ADDRESS)

    // Get contract instances
    tellor = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", TELLOR_MASTER)
    flex = await ethers.getContractAt("tellorflex/contracts/TellorFlex.sol:TellorFlex", TELLORFLEX)
    // parachute = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", PARACHUTE, devWallet)

    // Deploy TellorDataBridge
    blobstream = await ethers.deployContract("TellorDataBridge", [DEV_WALLET, VALIDATOR_SET_DOMAIN_SEPARATOR_SEPOLIA])
    fakeValCheckpoint = ethers.utils.solidityKeccak256(["string"], ["testy"])
    await blobstream.init(1, 2, UNBONDING_PERIOD, fakeValCheckpoint)

    // Deploy TokenBridge
    tbridge = await ethers.deployContract("TokenBridge", [TELLOR_MASTER, blobstream.address, TELLORFLEX])

    // Fund accounts
    await accounts[10].sendTransaction({
      to: BIGWALLET,
      value: ethers.utils.parseEther("10.0")
    })
    await accounts[10].sendTransaction({
      to: DEITY_ADDRESS,
      value: ethers.utils.parseEther("2.0")
    })
    await accounts[10].sendTransaction({
      to: DEV_WALLET,
      value: ethers.utils.parseEther("2.0")
    })

    // Confirm current oracle recipient is the old token bridge
    const ORACLE_CONTRACT_KEY = ethers.utils.keccak256(ethers.utils.toUtf8Bytes("_ORACLE_CONTRACT"))
    const initialOracle = await tellor.getAddressVars(ORACLE_CONTRACT_KEY)
    expect(initialOracle).to.equal(OLD_TOKEN_BRIDGE)

    // Deploy UpdateOracleTestnet pointing at the NEW TokenBridge
    const updateOracleLogic = await ethers.deployContract("UpdateOracleTestnet", [tbridge.address])

    // Use deity to point TellorMaster implementation to UpdateOracleTestnet
    const masterAdmin = await ethers.getContractAt(
      "contracts/tellor360/oldContracts/contracts/tellor3/TellorMaster.sol:TellorMaster",
      TELLOR_MASTER,
      deityWallet
    )
    await masterAdmin.changeTellorContract(updateOracleLogic.address)

    // Call init() via the proxy so new token bridge is set as the oracle contract at tellor master
    const updateOracleViaProxy = await ethers.getContractAt(
      "contracts/testing/UpdateOracleTestnet.sol:UpdateOracleTestnet",
      TELLOR_MASTER,
      devWallet
    )
    await updateOracleViaProxy.init()

    // Point implementation back to Tellor360
    await masterAdmin.changeTellorContract(TELLOR_360_CONTRACT)
  })

  it("Oracle address change works", async function() {
    const ORACLE_CONTRACT_KEY = h.hash(ethers.utils.toUtf8Bytes("_ORACLE_CONTRACT"))
    const newOracle = await tellor.getAddressVars(ORACLE_CONTRACT_KEY)
    expect(newOracle).to.equal(tbridge.address)

    // master is pointing back to Tellor360 implementation
    const TELLOR_CONTRACT_KEY = h.hash(ethers.utils.toUtf8Bytes("_TELLOR_CONTRACT"))
    const currentImpl = await tellor.getAddressVars(TELLOR_CONTRACT_KEY)
    expect(currentImpl).to.equal(TELLOR_360_CONTRACT)
  })

  it("New token bridge receives minted tokens", async function() {
    const balanceBefore = await tellor.balanceOf(tbridge.address)
    // ensure time passes so something mints
    await h.advanceTime(86400)
    await tellor.mintToOracle()
    const balanceAfter = await tellor.balanceOf(tbridge.address)
    assert(balanceAfter > balanceBefore, "balance after should be greater than balance before")
  })
})