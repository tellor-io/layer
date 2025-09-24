const { expect } = require("chai");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const { ethers } = require("hardhat");

describe.skip("Testnet Oracle Address Change - Real Deployed Contracts Forking Tests", function() {
  // Sepolia addresses
  const TELLOR_MASTER = "0x80fc34a2f9FfE86F41580F47368289C402DEc660"
  const DEV_WALLET = "0x34Fae97547E990ef0E05e05286c51E4645bf1A85"
  const BIGWALLET = "0x20bEC8F31dea6C13A016DC7fCBdF74f61DC8Ec2c"
  const TELLORFLEX = "0xB19584Be015c04cf6CFBF6370Fe94a58b7A38830"
  const OLD_TOKEN_BRIDGE = "0x5acb5977f35b1A91C4fE0F4386eB669E046776F2"
  const UNBONDING_PERIOD = 86400 * 7 * 3 // 3 weeks layer unbonding period
  const DEITY_ADDRESS = "0x20bEC8F31dea6C13A016DC7fCBdF74f61DC8Ec2c"
  const TELLOR_360_CONTRACT = "0x726737F28EA0BA5D23e16d1C3bb852982ff8651A" // current TELLOR_CONTRACT

  // Deployed contract addresses
  const DEPLOYED_TOKEN_BRIDGE = "0x87e025f9c3E20E8Cd1a5D2854237D75A4624F72e"
  const DEPLOYED_DATA_BRIDGE = "0x685534aae0171E541fC9AD41b0C3444275262264"
  const DEPLOYED_UPDATE_ORACLE = "0x928707dd5341EAe39cc21ac70161b3DE6f24839e"
  const DEPLOYED_TOKEN_BRIDGE_DEPLOYER = "0xfE2952AD10262C6b466070CA34dBB7fA54b882e3"

  const EVM_RECIPIENT = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"
  const LAYER_RECIPIENT = "tellor1zy50vdk8fdae0var2ryjhj2ysxtcm8dp2qtckd"

  const abiCoder = new ethers.utils.AbiCoder()
  let valset_domain_sep_args = abiCoder.encode(["string", "string"], ["checkpoint", "layertest-4"])
  const VALIDATOR_SET_DOMAIN_SEPARATOR_SEPOLIA = h.hash(valset_domain_sep_args)

  let accounts = null
  let flex = null
  let tellor = null
  let devWallet = null
  let deityWallet = null
  let tbridge = null
  let tokenBridgeDeployerWallet = null
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
    await h.impersonateAccount(DEPLOYED_TOKEN_BRIDGE_DEPLOYER)

    // Get accounts
    accounts = await ethers.getSigners()
    devWallet = await ethers.provider.getSigner(DEV_WALLET)
    bigWallet = await ethers.provider.getSigner(BIGWALLET)
    deityWallet = await ethers.provider.getSigner(DEITY_ADDRESS)
    tokenBridgeDeployerWallet = await ethers.provider.getSigner(DEPLOYED_TOKEN_BRIDGE_DEPLOYER)

    // Get contract instances
    tellor = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", TELLOR_MASTER)
    flex = await ethers.getContractAt("tellorflex/contracts/TellorFlex.sol:TellorFlex", TELLORFLEX)
    // parachute = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", PARACHUTE, devWallet)

    // Deploy TellorDataBridge
    blobstream = await ethers.getContractAt("TellorDataBridge", DEPLOYED_DATA_BRIDGE)
    fakeValCheckpoint = ethers.utils.solidityKeccak256(["string"], ["testy"])
    // await blobstream.connect(tokenBridgeDeployerWallet).init(1, 2, UNBONDING_PERIOD, fakeValCheckpoint)

    // Deploy TokenBridge
    tbridge = await ethers.getContractAt("TokenBridge", DEPLOYED_TOKEN_BRIDGE)

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
    const updateOracleLogic = await ethers.getContractAt("UpdateOracleTestnet", DEPLOYED_UPDATE_ORACLE)

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

  it("data bridge works", async function () {
    val1 = ethers.Wallet.createRandom()
    val2 = ethers.Wallet.createRandom()
    initialValAddrs = [val1.address, val2.address]
    initialPowers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    threshold = 2
    blocky = await h.getBlock()
    valTimestamp = (blocky.timestamp - 2) * 1000
    newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
    valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp, VALIDATOR_SET_DOMAIN_SEPARATOR_SEPOLIA)

    await blobstream.connect(tokenBridgeDeployerWallet).init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint)

    querydata = abiCoder.encode(["string"], ["myquery"])
    queryId = h.hash(querydata)
    value = abiCoder.encode(["uint256"], [2000])
    blocky = await h.getBlock()
    timestamp = (blocky.timestamp - 2) * 1000
    aggregatePower = 3
    attestTimestamp = timestamp + 1000
    previousTimestamp = 0
    nextTimestamp = 0
    lastConsensusTimestamp = timestamp
    newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
    valCheckpoint = await h.calculateValCheckpoint(newValHash, threshold, valTimestamp, VALIDATOR_SET_DOMAIN_SEPARATOR_SEPOLIA)
    dataDigest = await h.getDataDigest(
        queryId,
        value,
        timestamp,
        aggregatePower,
        previousTimestamp,
        nextTimestamp,
        valCheckpoint,
        attestTimestamp,
        lastConsensusTimestamp
    )
    currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
    sig1 = await h.layerSign(dataDigest, val1.privateKey)
    sig2 = await h.layerSign(dataDigest, val2.privateKey)
    sigStructArray = await h.getSigStructArray([sig1, sig2])
    oracleDataStruct = await h.getOracleDataStruct(
        queryId,
        value,
        timestamp,
        aggregatePower,
        previousTimestamp,
        nextTimestamp,
        attestTimestamp,
        lastConsensusTimestamp
    )
    await blobstream.verifyOracleData(
        oracleDataStruct,
        currentValSetArray,
        sigStructArray
    )
  })

  it("token bridge works", async function() {
    // initialize data bridge
    val1 = ethers.Wallet.createRandom()
    val2 = ethers.Wallet.createRandom()
    initialValAddrs = [val1.address, val2.address]
    initialPowers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    threshold = 2
    blocky = await h.getBlock()
    valTimestamp = (blocky.timestamp - 2) * 1000
    newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
    valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp, VALIDATOR_SET_DOMAIN_SEPARATOR_SEPOLIA)

    await blobstream.connect(tokenBridgeDeployerWallet).init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint)
    // test only deployer can initialize
    await h.expectThrow(tbridge.connect(accounts[1]).init(5, 3)) // not deployer
    
    // test successful initialization
    assert.equal(await tbridge.initialized(), false, "should not be initialized yet")
    assert.equal(await tbridge.depositId(), 0, "depositId should be 0 initially")
    
    await tbridge.connect(tokenBridgeDeployerWallet).init(5, 3)
    
    // verify initialization state
    assert.equal(await tbridge.initialized(), true, "should be initialized")
    assert.equal(await tbridge.depositId(), 5, "depositId should be set correctly")
    
    // verify withdraw claims are set correctly
    assert.equal(await tbridge.withdrawClaimed(0), true, "withdrawId 0 should be claimed")
    assert.equal(await tbridge.withdrawClaimed(1), true, "withdrawId 1 should be claimed")
    assert.equal(await tbridge.withdrawClaimed(2), true, "withdrawId 2 should be claimed")
    assert.equal(await tbridge.withdrawClaimed(3), false, "withdrawId 3 should not be claimed")
    assert.equal(await tbridge.withdrawClaimed(4), false, "withdrawId 4 should not be claimed")
    
    // test cannot initialize twice
    await h.expectThrow(tbridge.init(10, 5)) // already initialized

    // fund the fresh bridge with tokens for testing
    await tellor.connect(bigWallet).transfer(tbridge.address, h.toWei("1000"))

    // test deposit with higher starting depositId
    depositAmount = h.toWei("2")
    tip = h.toWei("0")
    await tellor.connect(bigWallet).approve(tbridge.address, h.toWei("100"))
    await tbridge.connect(bigWallet).depositToLayer(depositAmount, tip, LAYER_RECIPIENT)
    
    // verify deposit worked with correct incremented ID
    assert.equal(await tbridge.depositId(), 6, "depositId should increment from 5 to 6")
    depositDetails = await tbridge.deposits(6)
    assert.equal(depositDetails.amount.toString(), depositAmount, "deposit amount should be correct")
    assert.equal(depositDetails.recipient, LAYER_RECIPIENT, "deposit recipient should be correct")
    assert.equal(depositDetails.sender, BIGWALLET, "deposit sender should be correct")

    // test withdraw with higher starting withdrawId
    await h.advanceTime(43200)
    value = h.getWithdrawValue(EVM_RECIPIENT, LAYER_RECIPIENT, 20)
    blocky = await h.getBlock()
    timestamp = (blocky.timestamp - 43200) * 1000
    aggregatePower = 3
    attestTimestamp = blocky.timestamp * 1000
    previousTimestamp = 0
    nextTimestamp = 0
    lastConsensusTimestamp = timestamp
    
    // create withdraw for depositId 4 (which should not be claimed)
    WITHDRAW4_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, 4])
    WITHDRAW4_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW4_QUERY_DATA_ARGS])
    WITHDRAW4_QUERY_ID = h.hash(WITHDRAW4_QUERY_DATA)
    
    dataDigest = await h.getDataDigest(
        WITHDRAW4_QUERY_ID,
        value,
        timestamp,
        aggregatePower,
        previousTimestamp,
        nextTimestamp,
        valCheckpoint,
        attestTimestamp,
        lastConsensusTimestamp
    )
    currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
    sig1 = await h.layerSign(dataDigest, val1.privateKey)
    sig2 = await h.layerSign(dataDigest, val2.privateKey)
    sigStructArray = await h.getSigStructArray([sig1, sig2])
    oracleDataStruct = await h.getOracleDataStruct(
        WITHDRAW4_QUERY_ID,
        value,
        timestamp,
        aggregatePower,
        previousTimestamp,
        nextTimestamp,
        attestTimestamp,
        lastConsensusTimestamp
    )
    
    await tbridge.withdrawFromLayer(
        oracleDataStruct,
        currentValSetArray,
        sigStructArray,
        4
    )
    
    // verify withdraw worked
    recipientBal = await tellor.balanceOf(EVM_RECIPIENT)
    expectedBal = 20e12 // 20 loya converted to wei
    assert.equal(recipientBal.toString(), expectedBal, "recipient balance should be correct")
    assert.equal(await tbridge.withdrawClaimed(4), true, "withdrawId 4 should now be claimed")
  })
})