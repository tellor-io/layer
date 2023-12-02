const { expect } = require("chai");
const h = require("./helpers/helpers");
const hre = require("hardhat");
var assert = require('chai').assert;
const web3 = require('web3');
const { ethers } = require("hardhat");

describe("Function Tests - TellorLayerTransition", function() {

  const tellorMaster = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"
  const DEV_WALLET = "0x39E419bA25196794B595B2a595Ea8E527ddC9856"
  const PARACHUTE = "0x83eB2094072f6eD9F57d3F19f54820ee0BaE6084"
  const BIGWALLET = "0xF977814e90dA44bFA03b6295A0616a897441aceC";
  const TELLOR_FLEX = "0x8cFc184c877154a8F9ffE0fe75649dbe5e2DBEbf"
  const abiCoder = new ethers.utils.AbiCoder();
  const keccak256 = web3.utils.keccak256;
  const ETH_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["eth", "usd"]);
  const ETH_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", ETH_QUERY_DATA_ARGS]);
  const ETH_QUERY_ID = web3.utils.keccak256(ETH_QUERY_DATA);


  let accounts = null
  let oracle = null
  let tellor = null
  let newGovernance = null
  let governance = null
  let controller = null
  let cfac,ofac,tfac,gfac,parachute,govBig,govTeam
  let govSigner = null
  let devWallet = null
  let bigWallet = null
  let reporter = null
  let flex = null
  let tbridge, dbridge
  let transition

  beforeEach("deploy and setup Tellor360", async function() {

    await hre.network.provider.request({
      method: "hardhat_reset",
      params: [{forking: {
            jsonRpcUrl: hre.config.networks.hardhat.forking.url,
            blockNumber:18678330

          },},],
      });

    await hre.network.provider.request({
        method: "hardhat_impersonateAccount",
        params: [BIGWALLET]}
    )
    // })

    //account forks
    accounts = await ethers.getSigners()
    reporter = accounts[10]
    devWallet = await ethers.provider.getSigner(DEV_WALLET);
    bigWallet = await ethers.provider.getSigner(BIGWALLET);



    //contract forks
    tellor = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", tellorMaster)
    flex = await ethers.getContractAt("TellorFlex", TELLOR_FLEX)

    // deploy new contracts
    DataBridgeFac = await ethers.getContractFactory("BridgeMock")
    dbridge = await DataBridgeFac.deploy()
    await dbridge.deployed()
    TokenBridgeFac = await ethers.getContractFactory("TokenBridge")
    tbridge = await TokenBridgeFac.deploy(tellor.address, dbridge.address)
    await tbridge.deployed()
    TransitionFac = await ethers.getContractFactory("TellorLayerTransition")
    transition = await TransitionFac.deploy(tellor.address, tbridge.address, dbridge.address, flex.address)
    await transition.deployed()
    
    // stake reporter to flex
    reporterStakeAmt = h.toWei("1000")
    await tellor.connect(bigWallet).transfer(reporter.address, reporterStakeAmt)
    await tellor.connect(reporter).approve(flex.address, reporterStakeAmt)
    await flex.connect(reporter).depositStake(reporterStakeAmt)
    

  });

  it("constructor()", async function() {
    assert(await transition.token() == tellor.address, "token address should be set")
    assert(await transition.tokenBridge() == tbridge.address, "bridge address should be set")
    assert(await transition.dataBridge() == dbridge.address, "dataBridge address should be set")
  })

  it("mintToOracle", async function() {
    ORACLE_ADDR_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"]);
    ORACLE_ADDR_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_QUERY_DATA_ARGS]);
    ORACLE_ADDR_QUERY_ID = web3.utils.keccak256(ORACLE_ADDR_QUERY_DATA);
    VALUE = abiCoder.encode(["address"], [transition.address]);
    await flex.connect(reporter).submitValue(ORACLE_ADDR_QUERY_ID, VALUE, 0, ORACLE_ADDR_QUERY_DATA);
    await h.advanceTime(43200 + 1)
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 8)
    await tellor.updateOracleAddress()
    
    transitionBalBefore = await tellor.balanceOf(transition.address)
    assert(transitionBalBefore.eq(0), "transition contract should have 0 TRB")
    await tellor.mintToOracle()
    transitionBalAfter = await tellor.balanceOf(transition.address)
    assert(transitionBalAfter > 0, "transition contract should have TRB")
  })

  it("transferToTokenBridge", async function() {
    ORACLE_ADDR_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"]);
    ORACLE_ADDR_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_QUERY_DATA_ARGS]);
    ORACLE_ADDR_QUERY_ID = web3.utils.keccak256(ORACLE_ADDR_QUERY_DATA);
    VALUE = abiCoder.encode(["address"], [transition.address]);
    await flex.connect(reporter).submitValue(ORACLE_ADDR_QUERY_ID, VALUE, 0, ORACLE_ADDR_QUERY_DATA);
    await h.advanceTime(43200 + 1)
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 8)
    await tellor.updateOracleAddress()

    transitionBalBefore = await tellor.balanceOf(transition.address)
    tBridgeBalBefore = await tellor.balanceOf(tbridge.address)
    assert(transitionBalBefore.eq(0), "transition contract should have 0 TRB")
    assert(tBridgeBalBefore.eq(0), "bridge contract should have 0 TRB")
    await transition.transferToTokenBridge()
    transitionBalAfter = await tellor.balanceOf(transition.address)
    tBridgeBalAfter = await tellor.balanceOf(tbridge.address)
    assert(transitionBalAfter == 0, "transition contract should have 0 TRB")
    assert(tBridgeBalAfter > 0, "bridge contract should have TRB")
  })

  it("getDataBefore", async function() {
    ORACLE_ADDR_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"]);
    ORACLE_ADDR_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_QUERY_DATA_ARGS]);
    ORACLE_ADDR_QUERY_ID = web3.utils.keccak256(ORACLE_ADDR_QUERY_DATA);
    VALUE = abiCoder.encode(["address"], [transition.address]);
    await flex.connect(reporter).submitValue(ORACLE_ADDR_QUERY_ID, VALUE, 0, ORACLE_ADDR_QUERY_DATA);
    await h.advanceTime(43200 + 1)
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 8)
    await tellor.updateOracleAddress()

    blocky1 = await h.getBlock()
    flexVal = await flex.getDataBefore(ETH_QUERY_ID, blocky1.timestamp)
    decodedFlexVal = abiCoder.decode(["uint256"], flexVal._value)
    assert.isAbove(flexVal._timestampRetrieved, 0, "timestamp should be greater than block timestamp")
    assert.isAbove(decodedFlexVal[0], 0, "value should be greater than 0")

    layerTransitionVal = await transition.getDataBefore(ETH_QUERY_ID, blocky1.timestamp)
    decodedLayerTransitionVal = abiCoder.decode(["uint256"], layerTransitionVal._value)
    assert.equal(layerTransitionVal._timestampRetrieved.toNumber(), flexVal._timestampRetrieved.toNumber(), "timestamp should be equal")
    assert.isTrue(decodedLayerTransitionVal[0].eq(decodedFlexVal[0]), "value should be equal")
  })

  it("getIndexForDataBefore", async function() {
    ORACLE_ADDR_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"]);
    ORACLE_ADDR_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_QUERY_DATA_ARGS]);
    ORACLE_ADDR_QUERY_ID = web3.utils.keccak256(ORACLE_ADDR_QUERY_DATA);
    VALUE = abiCoder.encode(["address"], [transition.address]);
    await flex.connect(reporter).submitValue(ORACLE_ADDR_QUERY_ID, VALUE, 0, ORACLE_ADDR_QUERY_DATA);
    await h.advanceTime(43200 + 1)
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 8)
    await tellor.updateOracleAddress()

    blocky1 = await h.getBlock()
    flexVal = await flex.getIndexForDataBefore(ETH_QUERY_ID, blocky1.timestamp)
    assert.isAbove(flexVal._index.toNumber(), 0, "timestamp should be greater than block timestamp")

    layerTransitionVal = await transition.getIndexForDataBefore(ETH_QUERY_ID, blocky1.timestamp)
    assert.equal(layerTransitionVal._index.toNumber(), flexVal._index.toNumber(), "timestamp should be equal")
  })

  it("getNewValueCountbyQueryId", async function() {
    ORACLE_ADDR_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"]);
    ORACLE_ADDR_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_QUERY_DATA_ARGS]);
    ORACLE_ADDR_QUERY_ID = web3.utils.keccak256(ORACLE_ADDR_QUERY_DATA);
    VALUE = abiCoder.encode(["address"], [transition.address]);
    await flex.connect(reporter).submitValue(ORACLE_ADDR_QUERY_ID, VALUE, 0, ORACLE_ADDR_QUERY_DATA);
    await h.advanceTime(43200 + 1)
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 8)
    await tellor.updateOracleAddress()

    blocky1 = await h.getBlock()
    flexVal = await flex.getNewValueCountbyQueryId(ETH_QUERY_ID)
    assert.isAbove(flexVal.toNumber(), 0, "timestamp should be greater than block timestamp")

    layerTransitionVal = await transition.getNewValueCountbyQueryId(ETH_QUERY_ID)
    assert.equal(layerTransitionVal.toNumber(), flexVal.toNumber(), "timestamp should be equal")
  })

  it("getReporterByTimestamp", async function() {
    ORACLE_ADDR_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"]);
    ORACLE_ADDR_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_QUERY_DATA_ARGS]);
    ORACLE_ADDR_QUERY_ID = web3.utils.keccak256(ORACLE_ADDR_QUERY_DATA);
    VALUE = abiCoder.encode(["address"], [transition.address]);
    await flex.connect(reporter).submitValue(ORACLE_ADDR_QUERY_ID, VALUE, 0, ORACLE_ADDR_QUERY_DATA);
    blocky1 = await h.getBlock()
    await h.advanceTime(43200 + 1)
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 8)
    await tellor.updateOracleAddress()
    
    flexVal = await flex.getReporterByTimestamp(ORACLE_ADDR_QUERY_ID, blocky1.timestamp)
    assert.equal(flexVal, reporter.address, "timestamp should be greater than block timestamp")

    layerTransitionVal = await transition.getReporterByTimestamp(ORACLE_ADDR_QUERY_ID, blocky1.timestamp)
    assert.equal(layerTransitionVal, reporter.address, "timestamp should be equal")
  })
})