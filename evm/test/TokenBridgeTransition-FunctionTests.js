// const { AbiCoder } = require("@ethersproject/abi");
const { expect } = require("chai");
const h = require("./helpers/evmHelpers");

describe("Function Tests - NewTransition", function() {

  const TELLOR_MASTER = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"
  const DEV_WALLET = "0x39E419bA25196794B595B2a595Ea8E527ddC9856"
  const PARACHUTE = "0x83eB2094072f6eD9F57d3F19f54820ee0BaE6084"
  const BIGWALLET = "0x5a52E96BAcdaBb82fd05763E25335261B270Efcb";
  const GOVERNANCE_FLEX = "0xB30b1B98d8276b80bC4f5aF9f9170ef3220EC27D"
  const TELLORFLEX = "0x8cFc184c877154a8F9ffE0fe75649dbe5e2DBEbf"
  const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks layer unbonding period
  const abiCoder = new ethers.AbiCoder();
  const ETH_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["eth", "usd"]);
  const ETH_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", ETH_QUERY_DATA_ARGS]);
  const ETH_QUERY_ID = h.hash(ETH_QUERY_DATA);
  const ORACLE_ADDR_UPDATE_QUERY_DATA_ARGS = abiCoder.encode(["bytes"], ["0x"])
  const ORACLE_ADDR_UPDATE_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TellorOracleAddress", ORACLE_ADDR_UPDATE_QUERY_DATA_ARGS])
  const ORACLE_ADDR_UPDATE_QUERY_ID = h.hash(ORACLE_ADDR_UPDATE_QUERY_DATA);

  let accounts = null
  let flex = null
  let tellor = null
  let govflex = null
  let devWallet = null
  let blocky0 = null
  let blocky1 = null
  let blocky2 = null
  let snapshot = null

  before(async function() {
    // take hardhat network snapshot
    snapshot = await h.takeSnapshot()
  })

  beforeEach("deploy and transition to Layer TokenBridge", async function() {
    // restore from snapshot
    await snapshot.restore()
    await h.impersonateAccount(BIGWALLET)

    //account forks
    accounts = await ethers.getSigners()
    devWallet = await ethers.provider.getSigner(DEV_WALLET);
    bigWallet = await ethers.provider.getSigner(BIGWALLET);

    //contract forks
    tellor = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor", TELLOR_MASTER)
    govflex = await ethers.getContractAt("polygongovernance/contracts/Governance.sol:Governance", GOVERNANCE_FLEX)
    flex = await ethers.getContractAt("tellorflex/contracts/TellorFlex.sol:TellorFlex", TELLORFLEX)
    parachute = await ethers.getContractAt("contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor",PARACHUTE, devWallet);

    // get blobstream initial params
    // valTs = await h.getValsetTimestampByIndex(0)
    // valParams = await h.getValsetCheckpointParams(valTs)
    // valSet = await h.getValset(valParams.timestamp)
    // deploy blobstream (no need for actual queries)
    blobstream = await ethers.deployContract(
      "BlobstreamO", [
       1,
       2,
      UNBONDING_PERIOD,
      h.hash("testy"),
      DEV_WALLET
    ]
    )
    // deploy tokenbridge
    tbridge = await ethers.deployContract("TokenBridge", [TELLOR_MASTER,await blobstream.getAddress(), TELLORFLEX])
    // stake reporter
    await tellor.connect(bigWallet).transfer(await accounts[0].getAddress(), h.toWei("1000"))
    await tellor.connect(accounts[0]).approve(TELLORFLEX, h.toWei("1000"))
    await flex.connect(accounts[0]).depositStake(h.toWei("1000"))
    // report new oracle address
    newOracleAddrReport = abiCoder.encode(["address"], [await tbridge.getAddress()])
    await flex.connect(accounts[0]).submitValue(ORACLE_ADDR_UPDATE_QUERY_ID, newOracleAddrReport, 0, ORACLE_ADDR_UPDATE_QUERY_DATA)
    await h.advanceTime(43201)
    // update oracle address
    await tellor.updateOracleAddress()
    await h.advanceTime(86400 * 7)
    await tellor.updateOracleAddress()

    // submit some data
    await flex.submitValue(ETH_QUERY_ID, h.uintTob32("100"), 0, ETH_QUERY_DATA)
    blocky0 = await h.getBlock()
    await h.advanceTime(43201)
    await flex.submitValue(ETH_QUERY_ID, h.uintTob32("101"), 0, ETH_QUERY_DATA)
    blocky1 = await h.getBlock()
    await h.advanceTime(43201)
    await flex.submitValue(ETH_QUERY_ID, h.uintTob32("102"), 0, ETH_QUERY_DATA)
    blocky2 = await h.getBlock()

    // sleep 1 second for api rate limit
    await new Promise(r => setTimeout(r, 1000));
    this.timeout(100000)
  });

  it("constructor", async function() {
    // check if new oracle address is set
    expect(await tbridge.token() == tellor.getAddress(), "tellor should be set")
    expect(await tbridge.tellorFlex() == flex.getAddress(), "tellor should be set")
  })
  it("transition worked", async function() {
    // check if new oracle address is set
    expect(await tellor.getAddressVars(h.hash("_ORACLE_CONTRACT"))).to.equal(await tbridge.getAddress())
  })

  it("addStakingRewards()", async function () {
    await tellor.connect(bigWallet).transfer(await accounts[0].getAddress(), h.toWei("1"))
    await h.expectThrow(tbridge.connect(accounts[0]).addStakingRewards(h.toWei("1"))) // not approved
    await tellor.connect(accounts[0]).approve(await tbridge.getAddress(), h.toWei("1"))
    await tbridge.connect(accounts[0]).addStakingRewards(h.toWei("1"))
    expect(await tellor.balanceOf(await tbridge.getAddress())).to.equal(h.toWei("1"))
  })

  it("getDataBefore()", async function () {
    dataBefore = await tbridge.getDataBefore(ETH_QUERY_ID, blocky1.timestamp)
    expect(dataBefore[0]).to.equal(true)
    expect(dataBefore[1]).to.equal(h.uintTob32("100"))
    expect(dataBefore[2]).to.equal(blocky0.timestamp)

    dataBefore = await tbridge.getDataBefore(ETH_QUERY_ID, blocky2.timestamp)
    expect(dataBefore[0]).to.equal(true)
    expect(dataBefore[1]).to.equal(h.uintTob32("101"))
    expect(dataBefore[2]).to.equal(blocky1.timestamp)

    // check for updateOracleAddress query id
    dataBefore = await tbridge.getDataBefore(ORACLE_ADDR_UPDATE_QUERY_ID, blocky2.timestamp)
    blocky = await h.getBlock()
    expect(dataBefore[0]).to.equal(true)
    expect(dataBefore[1]).to.equal(abiCoder.encode(["address"], [await tbridge.getAddress()]))
    expect(dataBefore[2]).to.equal(blocky.timestamp)

    // submit different oracle address
    await h.advanceTime(43200)
    badOracleAddrReport = abiCoder.encode(["address"], [await accounts[1].getAddress()])
    await flex.connect(accounts[0]).submitValue(ORACLE_ADDR_UPDATE_QUERY_ID, badOracleAddrReport, 0, ORACLE_ADDR_UPDATE_QUERY_DATA)
    blocky = await h.getBlock()
    dataBefore = await tbridge.getDataBefore(ORACLE_ADDR_UPDATE_QUERY_ID, blocky.timestamp + 100)
    blocky = await h.getBlock()
    expect(dataBefore[0]).to.equal(true)
    expect(dataBefore[1]).to.equal(abiCoder.encode(["address"], [await tbridge.getAddress()]))
    expect(dataBefore[2]).to.equal(blocky.timestamp)
  })

  it("getIndexForDataBefore()", async function () {
    indexBefore = await tbridge.getIndexForDataBefore(ETH_QUERY_ID, blocky0.timestamp)
    expect(indexBefore[0]).to.equal(true)
    expect(indexBefore[1]).to.be.greaterThan(0)

    indexBefore1 = await tbridge.getIndexForDataBefore(ETH_QUERY_ID, blocky1.timestamp)
    expect(indexBefore1[0]).to.equal(true)
    expect(indexBefore1[1]).to.equal(BigInt(indexBefore[1]) + BigInt(1))

    indexBefore2 = await tbridge.getIndexForDataBefore(ETH_QUERY_ID, blocky2.timestamp)
    expect(indexBefore2[0]).to.equal(true)
    expect(indexBefore2[1]).to.equal(BigInt(indexBefore1[1]) + BigInt(1))
  })

  it("getNewValueCountbyQueryId()", async function () {
    count = await tbridge.getNewValueCountbyQueryId(ETH_QUERY_ID)
    expect(count).to.be.greaterThan(0)

    await h.advanceTime(43200)
    await flex.submitValue(ETH_QUERY_ID, h.uintTob32("103"), 0, ETH_QUERY_DATA)
    count1 = await tbridge.getNewValueCountbyQueryId(ETH_QUERY_ID)
    expect(count1).to.equal(count + BigInt(1))
  })

  it("getReporterByTimestamp()", async function () {
    reporter = await tbridge.getReporterByTimestamp(ETH_QUERY_ID, blocky0.timestamp)
    expect(reporter).to.equal(await accounts[0].getAddress())
  })

  it("getTimestampbyQueryIdandIndex()", async function () {
    count = await tbridge.getNewValueCountbyQueryId(ETH_QUERY_ID)
    expect(count).to.be.greaterThan(0)

    timestamp = await tbridge.getTimestampbyQueryIdandIndex(ETH_QUERY_ID, count - BigInt(1))
    expect(timestamp).to.equal(blocky2.timestamp)
  })

  it("getTimeOfLastNewValue()", async function () {
    time = await tbridge.getTimeOfLastNewValue()
    expect(time).to.equal(blocky2.timestamp)
  })

  it("isInDispute()", async function () {
    expect(await tbridge.isInDispute(ETH_QUERY_ID, blocky2.timestamp)).to.equal(false)
    await tellor.connect(bigWallet).approve(GOVERNANCE_FLEX, h.toWei("100"))
    await govflex.connect(bigWallet).beginDispute(ETH_QUERY_ID, blocky2.timestamp)
    expect(await tbridge.isInDispute(ETH_QUERY_ID, blocky2.timestamp)).to.equal(true)
  })

  it("verify()", async function () {
    expect(await tbridge.verify()).to.equal(9999)
  })

  it("mintToOracle()", async function () {
    expect(await tellor.balanceOf(await tbridge.getAddress())).to.equal(0)
    await tellor.mintToOracle()
    expect(await tellor.balanceOf(await tbridge.getAddress())).to.be.greaterThan(0)
    
  })
  it("mintToTeam()", async function () {
    expect(await tellor.balanceOf(await tbridge.getAddress())).to.equal(0)
    await tellor.mintToOracle()
    expect(await tellor.balanceOf(await tbridge.getAddress())).to.be.greaterThan(0)
    let teamBal = await tellor.balanceOf(DEV_WALLET)
    await tellor.mintToTeam()
    expect(await tellor.balanceOf(DEV_WALLET) > teamBal, "mint to team should work")
  })

})