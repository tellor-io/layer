const { expect } = require("chai");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const web3 = require('web3');
const { ethers } = require("hardhat");

describe("E2E Forking Tests - TokenBridge Transition", function() {
  // Mainnet addresses
  const TELLOR_MASTER = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"
  const DEV_WALLET = "0x39E419bA25196794B595B2a595Ea8E527ddC9856"
  const PARACHUTE = "0x83eB2094072f6eD9F57d3F19f54820ee0BaE6084"
  const BIGWALLET = "0x5a52E96BAcdaBb82fd05763E25335261B270Efcb"
  const GOVERNANCE_FLEX = "0xB30b1B98d8276b80bC4f5aF9f9170ef3220EC27D"
  const TELLORFLEX = "0x8cFc184c877154a8F9ffE0fe75649dbe5e2DBEbf"
  const LIQUITY_PRICE_FEED = "0x4c517D4e2C851CA76d7eC94B805269Df0f2201De"
  const TELLOR_PROVIDER_AMPL = "0xf5b7562791114fB1A8838A9E8025de4b7627Aa79"
  const MEDIAN_ORACLE_AMPL = "0x99C9775E076FDF99388C029550155032Ba2d8914"
  const UNBONDING_PERIOD = 86400 * 7 * 3 // 3 weeks layer unbonding period

  const abiCoder = new ethers.utils.AbiCoder()
  const ETH_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["eth", "usd"])
  const ETH_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", ETH_QUERY_DATA_ARGS])
  const ETH_QUERY_ID = h.hash(ETH_QUERY_DATA)
  const ORACLE_ADDR_UPDATE_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"])
  const ORACLE_ADDR_UPDATE_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_UPDATE_QUERY_DATA_ARGS])
  const ORACLE_ADDR_UPDATE_QUERY_ID = h.hash(ORACLE_ADDR_UPDATE_QUERY_DATA)
  const AMPL_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"])
  const AMPL_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["AmpleforthCustomSpotPrice", AMPL_QUERY_DATA_ARGS])
  const AMPL_QUERY_ID = h.hash(AMPL_QUERY_DATA)

  let accounts = null
  let flex = null
  let tellor = null
  let govflex = null
  let devWallet = null
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

    // Get accounts
    accounts = await ethers.getSigners()
    devWallet = await ethers.provider.getSigner(DEV_WALLET)
    bigWallet = await ethers.provider.getSigner(BIGWALLET)

    // Get contract instances
    tellor = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", TELLOR_MASTER)
    govflex = await ethers.getContractAt("polygongovernance/contracts/Governance.sol:Governance", GOVERNANCE_FLEX)
    flex = await ethers.getContractAt("tellorflex/contracts/TellorFlex.sol:TellorFlex", TELLORFLEX)
    parachute = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", PARACHUTE, devWallet)

    // Deploy TellorDataBridge
    blobstream = await ethers.deployContract("TellorDataBridge", [DEV_WALLET])
    fakeValCheckpoint = ethers.utils.solidityKeccak256(["string"], ["testy"])
    await blobstream.init(1, 2, UNBONDING_PERIOD, fakeValCheckpoint)

    // Deploy TokenBridge
    tbridge = await ethers.deployContract("TokenBridge", [TELLOR_MASTER, blobstream.address, TELLORFLEX])

    // Fund accounts
    await accounts[10].sendTransaction({
      to: BIGWALLET,
      value: ethers.utils.parseEther("10.0")
    })

    // Setup reporter
    await tellor.connect(bigWallet).transfer(accounts[0].address, h.toWei("1000"))
    await tellor.connect(accounts[0]).approve(TELLORFLEX, h.toWei("1000"))
    await flex.connect(accounts[0]).depositStake(h.toWei("1000"))

    // Report new oracle address
    const newOracleAddrReport = abiCoder.encode(["address"], [tbridge.address])
    await flex.connect(accounts[0]).submitValue(ORACLE_ADDR_UPDATE_QUERY_ID, newOracleAddrReport, 0, ORACLE_ADDR_UPDATE_QUERY_DATA)
    await h.advanceTime(43201)

    // Update oracle address
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 7)
    await tellor.updateOracleAddress()
  })

  it("Liquity reads from TokenBridge after transition", async function() {
    // Get Liquity price feed contract
    let liquityPriceFeed = await ethers.getContractAt("contracts/tellor360/testing/IPriceFeed.sol:IPriceFeed", LIQUITY_PRICE_FEED)

    // Submit ETH price through Flex
    await flex.connect(accounts[0]).submitValue(ETH_QUERY_ID, h.uintTob32(h.toWei("2095.15")), 0, ETH_QUERY_DATA)
    await h.advanceTime(60 * 15 + 1)

    // Verify Liquity reads correct price
    await liquityPriceFeed.fetchPrice()
    let lastGoodPrice = await liquityPriceFeed.lastGoodPrice()
    expect(BigInt(lastGoodPrice)).to.eq(BigInt("2095150000000000000000"), "Liquity ether price should be correct")

    // Submit updated price
    await h.advanceTime(60 * 60 * 12)
    await flex.connect(accounts[0]).submitValue(ETH_QUERY_ID, h.uintTob32(h.toWei("3395.16")), 0, ETH_QUERY_DATA)
    await h.advanceTime(60 * 15 + 1)

    // Verify Liquity reads updated price
    await liquityPriceFeed.fetchPrice()
    lastGoodPrice = await liquityPriceFeed.lastGoodPrice()
    expect(BigInt(lastGoodPrice)).to.eq(BigInt("3395160000000000000000"), "Liquity ether price should be correct")

    // Submit third price
    await h.advanceTime(60 * 60 * 12)
    await flex.connect(accounts[0]).submitValue(ETH_QUERY_ID, h.uintTob32(h.toWei("3395.17")), 0, ETH_QUERY_DATA)
    await h.advanceTime(60 * 15 + 1)

    // Verify Liquity reads third price
    await liquityPriceFeed.fetchPrice()
    lastGoodPrice = await liquityPriceFeed.lastGoodPrice()
    expect(BigInt(lastGoodPrice)).to.eq(BigInt("3395170000000000000000"), "Liquity ether price should be correct")
  })

  it("Liquity TellorCaller integration works correctly", async function() {
    // Deploy TellorCaller test contract
    const TellorCallerTest = await ethers.getContractFactory("contracts/tellor360/testing/liquity/TellorCaller.sol:TellorCaller")
    let tellorCallerTest = await TellorCallerTest.deploy(tellor.address)

    // Submit initial price
    await flex.connect(accounts[0]).submitValue(ETH_QUERY_ID, h.uintTob32(h.toWei("2095.15")), 0, ETH_QUERY_DATA)
    let blocky0 = await h.getBlock()
    await h.advanceTime(60 * 15 + 1)

    // Verify price through TellorCaller
    let currentVal = await tellorCallerTest.getTellorCurrentValue(1)
    assert(currentVal[0] == true, "ifRetrieve should be correct")
    assert(currentVal[1] == 2095150000, "current value should be correct")
    assert(currentVal[2] == blocky0.timestamp, "current timestamp should be correct")

    // Submit second price
    await h.advanceTime(60 * 60 * 12)
    await flex.connect(accounts[0]).submitValue(ETH_QUERY_ID, h.uintTob32(h.toWei("3395.16")), 0, ETH_QUERY_DATA)
    let blocky1 = await h.getBlock()

    // Verify price not updated before timeout
    currentVal = await tellorCallerTest.getTellorCurrentValue(1)
    assert(currentVal[0] == true, "ifRetrieve should be correct")
    assert(currentVal[1] == 2095150000, "current value should be correct")
    assert(currentVal[2] == blocky0.timestamp, "current timestamp should be correct")

    // Verify price updated after timeout
    await h.advanceTime(60 * 15 + 1)
    currentVal = await tellorCallerTest.getTellorCurrentValue(1)
    assert(currentVal[0] == true, "ifRetrieve should be correct")
    assert(currentVal[1] == 3395160000, "current value should be correct")
    assert(currentVal[2] == blocky1.timestamp, "current timestamp should be correct")
  })

  it("Ampleforth reads from TokenBridge after transition", async function() {
    // Get Ampleforth contract instances
    let tellorProviderAmpl = await ethers.getContractAt("contracts/tellor360/testing/TellorProvider.sol:TellorProvider", TELLOR_PROVIDER_AMPL)
    let medianOracleAmpl = await ethers.getContractAt("contracts/tellor360/testing/MedianOracle.sol:MedianOracle", MEDIAN_ORACLE_AMPL)

    // Submit first AMPL price through Flex
    await flex.connect(accounts[0]).submitValue(AMPL_QUERY_ID, h.uintTob32(h.toWei("1.23")), 0, AMPL_QUERY_DATA)
    let blocky0 = await h.getBlock()
    await h.advanceTime(86400)

    // Push value to Ampleforth provider
    await tellorProviderAmpl.pushTellor()

    // Verify correct timestamp pushed to provider
    let tellorReport = await tellorProviderAmpl.tellorReport()
    assert(tellorReport[0] == blocky0.timestamp || tellorReport[1] == blocky0.timestamp, "tellor report timestamp not pushed correctly")

    // Verify correct value pushed to median oracle
    let providerReports0 = await medianOracleAmpl.providerReports(tellorProviderAmpl.address, 0)
    let providerReports1 = await medianOracleAmpl.providerReports(tellorProviderAmpl.address, 1)
    assert(providerReports0.payload == h.toWei("1.23") || providerReports1.payload == h.toWei("1.23"), "tellor report value not pushed correctly")

    // Submit second AMPL price
    await flex.connect(accounts[0]).submitValue(AMPL_QUERY_ID, h.uintTob32(h.toWei("0.99")), 0, AMPL_QUERY_DATA)
    let blocky1 = await h.getBlock()
    await h.advanceTime(86400)

    // Push updated value
    await tellorProviderAmpl.pushTellor()

    // Verify updated timestamp
    tellorReport = await tellorProviderAmpl.tellorReport()
    assert(tellorReport[0] == blocky1.timestamp || tellorReport[1] == blocky1.timestamp, "updated tellor report timestamp not pushed")

    // Verify updated value
    providerReports0 = await medianOracleAmpl.providerReports(tellorProviderAmpl.address, 0)
    providerReports1 = await medianOracleAmpl.providerReports(tellorProviderAmpl.address, 1)
    assert(providerReports0.payload == h.toWei("0.99") || providerReports1.payload == h.toWei("0.99"), "updated tellor report value not pushed")
  })
})
