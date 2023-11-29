const {
  expect,
  assert
} = require("chai");
const { ethers } = require("hardhat");
const web3 = require('web3');
const h = require("./helpers/helpers");
const abiCoder = new ethers.utils.AbiCoder
const autopayQueryData = abiCoder.encode(["string", "bytes"], ["AutopayAddresses", abiCoder.encode(['bytes'], ['0x'])])
const autopayQueryId = ethers.utils.keccak256(autopayQueryData)
const TRB_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["trb", "usd"])
const TRB_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", TRB_QUERY_DATA_ARGS])
const TRB_QUERY_ID = ethers.utils.keccak256(TRB_QUERY_DATA)
const ETH_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["eth", "usd"])
const ETH_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", ETH_QUERY_DATA_ARGS])
const ETH_QUERY_ID = ethers.utils.keccak256(ETH_QUERY_DATA)
const MINIMUM_STAKE_AMOUNT = web3.utils.toWei("10")

describe("Governance End-To-End Tests", function() {

  let gov, flex, accounts, token, autopay, autopayArray;

  beforeEach(async function() {
    accounts = await ethers.getSigners();
    const Token = await ethers.getContractFactory("StakingToken");
    token = await Token.deploy();
    await token.deployed();
    const Governance = await ethers.getContractFactory("Governance");
    const TellorFlex = await ethers.getContractFactory("TellorFlex")
    flex = await TellorFlex.deploy(token.address, 86400/2, web3.utils.toWei("100"), web3.utils.toWei("10"), MINIMUM_STAKE_AMOUNT, TRB_QUERY_ID)
    await flex.deployed();
    gov = await Governance.deploy(flex.address,  accounts[0].address);
    await gov.deployed();
    await flex.init(gov.address)
    const Autopay = await ethers.getContractFactory("AutopayMock");
    autopay = await Autopay.deploy(token.address);
    await token.mint(accounts[1].address, web3.utils.toWei("1000"));
    autopayArray = abiCoder.encode(["address[]"], [[autopay.address]]);
  });
  it("Test multiple disputes", async function() {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("1000"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("100"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    await token.connect(accounts[1]).transfer(accounts[3].address, web3.utils.toWei("100"))
    await token.connect(accounts[1]).transfer(accounts[4].address, web3.utils.toWei("100"))
    await token.connect(accounts[1]).transfer(accounts[5].address, web3.utils.toWei("100"))
    await token.connect(accounts[4]).approve(flex.address, web3.utils.toWei("100"))
    await flex.connect(accounts[4]).depositStake(web3.utils.toWei("100"))
    await token.connect(accounts[5]).approve(flex.address, web3.utils.toWei("100"))
    await flex.connect(accounts[5]).depositStake(web3.utils.toWei("100"))
    let blocky = await h.getBlock()
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    let balance1 = await token.balanceOf(accounts[2].address)
    await h.expectThrow(gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) //no value exists
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await flex.connect(accounts[4]).submitValue(h.hash('0x123456'), h.bytes(200), 0, '0x123456')
    let blocky2 = await h.getBlock()
    await flex.connect(accounts[5]).submitValue(h.hash('0x1234'), h.bytes("a"), 0, '0x1234')
    let blocky3 = await h.getBlock()
    await h.expectThrow(gov.connect(accounts[4]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) // must have tokens to
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("30"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp);
    await gov.connect(accounts[2]).beginDispute(h.hash('0x123456'), blocky2.timestamp);
    await gov.connect(accounts[2]).beginDispute(h.hash('0x1234'), blocky3.timestamp);
    assert(await gov.voteCount() == 3, "vote count should be 3")
    let balance2 = await token.balanceOf(accounts[2].address)
    let vars = await gov.getDisputeInfo(1)
    let _hash = ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [ETH_QUERY_ID, blocky.timestamp])
    assert(vars[0] == ETH_QUERY_ID, "queryID should be correct")
    assert(vars[1] == blocky.timestamp, "timestamp should be correct")
    assert(vars[2] == h.bytes(100), "value should be correct")
    assert(vars[3] == accounts[1].address, "accounts[1] should be correct")
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 1, "open disputes on ID should be correct")
    assert(await gov.getVoteRounds(_hash) == 1, "number of vote rounds should be correct")
    vars = await gov.getDisputeInfo(2)
    _hash = await ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [h.hash('0x123456'), blocky2.timestamp])
    assert(vars[0] == h.hash('0x123456'), "queryID should be correct")
    assert(vars[1] == blocky2.timestamp, "timestamp should be correct")
    assert(vars[2] == h.bytes(200), "value should be correct")
    assert(vars[3] == accounts[4].address, "accounts[1] should be correct")
    assert(await gov.getOpenDisputesOnId(h.hash('0x123456')) == 1, "open disputes on ID should be correct")
    let vr = await gov.getVoteRounds(_hash)
    assert(vr.length == 1, "number of vote rounds should be correct")
    vars = await gov.getDisputeInfo(3)
    _hash = ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [h.hash('0x1234'), blocky3.timestamp])
    assert(vars[0] == h.hash('0x1234'), "queryID should be correct")
    assert(vars[1] == blocky3.timestamp, "timestamp should be correct")
    assert(vars[2] == h.bytes("a"), "value should be correct")
    assert(vars[3] == accounts[5].address, "accounts[1] should be correct")
    assert(await gov.getOpenDisputesOnId(h.hash('0x1234')) == 1, "open disputes on ID should be correct")
    vr = await gov.getVoteRounds(_hash)
    assert(vr.length == 1, "number of vote rounds should be correct")
    await h.advanceTime(86400 * 2);
    await gov.tallyVotes(1)
    await gov.tallyVotes(2)
    await gov.tallyVotes(3)
    for (var i = 1; i < 4; i++) {
      vars = await gov.getVoteInfo(i)

      assert(vars[3] == 2, "Vote result should be INVALID")
    }
    await h.advanceTime(86400 * 2);
    await gov.executeVote(1)
    await gov.executeVote(2)
    await gov.executeVote(3)
    for (var i = 1; i < 4; i++) {
      vars = await gov.getVoteInfo(i)
      assert(vars[3] == 2, "Vote result should be INVALID")
    }
    let s = await token.balanceOf(accounts[1].address)
    assert(s - web3.utils.toWei("510") == 0, "should have tokens returned")
  });
  it("Test no votes on a dispute", async function() {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("10"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    await h.advanceTime(86400)
    balance1 = await token.balanceOf(accounts[1].address)
    balance2 = await token.balanceOf(accounts[2].address)
    await gov.executeVote(1)
    assert(await token.balanceOf(accounts[1].address) - balance1 == web3.utils.toWei("10"), "account1 balance should increase by original stake amount")
    assert(await token.balanceOf(accounts[2].address) - balance2 == web3.utils.toWei("1"), "account2 balance should increase by fee amount")
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[3] == 2, "Vote result should be correct")
    let s = await token.balanceOf(accounts[1].address)
    assert(s - web3.utils.toWei("900") == 0, "should have tokens returned")
  });
  it("Test multiple vote rounds on a dispute, all passing", async function() {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("10"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    // Round 1
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(1, true, false)
    await gov.connect(accounts[2]).vote(1, true, false)
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    // Round 2
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("20"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(2, true, false)
    await gov.connect(accounts[2]).vote(2, true, false)
    await h.advanceTime(86400 * 4)
    await gov.tallyVotes(2)
    // Round 3
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("40"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(3, true, false)
    await gov.connect(accounts[2]).vote(3, true, false)
    await h.advanceTime(86400 * 6)
    await gov.tallyVotes(3)
    // Execute
    await h.advanceTime(86400 * 3)
    balance1 = await token.balanceOf(accounts[1].address)
    balance2 = await token.balanceOf(accounts[2].address)
    balanceGov = await token.balanceOf(gov.address)
    await h.expectThrow(gov.executeVote(1)) // Must be the final vote
    await h.expectThrow(gov.executeVote(2)) // Must be the final vote
    await gov.executeVote(3)
    await h.expectThrow(gov.executeVote(3)) // Vote has been executed
    assert(await token.balanceOf(accounts[1].address) - balance1 == 0, "account1 balance should not change")
    assert(await token.balanceOf(accounts[2].address) - balance2 == web3.utils.toWei("17"), "account2 balance should increase by original stake amount plus fee amount")
    assert(balanceGov - await token.balanceOf(gov.address) == web3.utils.toWei("17"), "governance balance should decrease by original stake amount plus fee amount")
    voteInfo = await gov.getVoteInfo(3)
    assert(voteInfo[3] == 1, "Vote result should be correct")
  });
  it("Test multiple vote rounds on a dispute,  overturn result", async function() {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("10"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    // Round 1
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("1"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(1, true, false)
    await gov.connect(accounts[2]).vote(1, true, false)
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    // Round 2
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("2"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(2, true, false)
    await gov.connect(accounts[2]).vote(2, true, false)
    await h.advanceTime(86400 * 4)
    await gov.tallyVotes(2)
    // Round 3
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("4"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(3, false, false)
    await gov.connect(accounts[2]).vote(3, false, false)
    await h.advanceTime(86400 * 6)
    await gov.tallyVotes(3)
    // Execute
    await h.advanceTime(86400 * 3)
    balance1 = await token.balanceOf(accounts[1].address)
    balance2 = await token.balanceOf(accounts[2].address)
    balanceGov = await token.balanceOf(gov.address)
    await h.expectThrow(gov.executeVote(1)) // Must be the final vote
    await h.expectThrow(gov.executeVote(2)) // Must be the final vote
    await gov.executeVote(3)
    await h.expectThrow(gov.executeVote(3)) // Vote has been executed
    assert(await token.balanceOf(accounts[1].address) - balance1 == web3.utils.toWei("17"), "account1 balance should increase by original stake amount plus fee amount")
    assert(await token.balanceOf(accounts[2].address) - balance2 == 0, "account2 balance should not change")
    assert(balanceGov - await token.balanceOf(gov.address) == web3.utils.toWei("17"), "governance balance should decrease by original stake amount plus fee amount")
    voteInfo = await gov.getVoteInfo(3)
    assert(voteInfo[3] == 0, "Vote result should be correct")
  });
  it("Test voting from all four stakeholder groups", async function() {
    // submit autopay addresses array to oracle
    await token.mint(accounts[19].address, web3.utils.toWei("10"))
    await token.connect(accounts[19]).approve(flex.address, web3.utils.toWei("10"))
    await flex.connect(accounts[19]).depositStake(web3.utils.toWei("10"))
    await flex.connect(accounts[19]).submitValue(autopayQueryId, autopayArray, 0, autopayQueryData)
    await h.advanceTime(86400)
    // define stakeholders
    user1 = accounts[9]
    user2 = accounts[10]
    reporter1 = accounts[2] // reporters are also tokenholders unless completely slashed
    reporter2 = accounts[3] // reporters are also tokenholders unless completely slashed
    tokenholder1 = accounts[1]
    tokenholder2 = accounts[4]
    multisig = accounts[0]
    // set user1
    await token.mint(user1.address, web3.utils.toWei("1"))
    await token.connect(user1).approve(autopay.address, web3.utils.toWei("1"))
    await autopay.connect(user1).tip(ETH_QUERY_ID, web3.utils.toWei("1"), ETH_QUERY_DATA)
    // // set user2
    await token.mint(user2.address, web3.utils.toWei("1"))
    await token.connect(user2).approve(autopay.address, web3.utils.toWei("1"))
    await autopay.connect(user2).tip(ETH_QUERY_ID, web3.utils.toWei("1"), ETH_QUERY_DATA)
    // set tokenholder2
    await token.connect(accounts[1]).transfer(tokenholder2.address, web3.utils.toWei("20"))
    // submit some reporter values
    await token.connect(accounts[1]).transfer(reporter1.address, web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(reporter2.address, web3.utils.toWei("10"))
    await token.connect(reporter1).approve(flex.address, web3.utils.toWei("10"))
    await token.connect(reporter2).approve(flex.address, web3.utils.toWei("10"))
    await flex.connect(reporter1).depositStake(web3.utils.toWei("10"))
    await flex.connect(reporter2).depositStake(web3.utils.toWei("10"))
    await flex.connect(reporter1).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await flex.connect(reporter2).submitValue(h.hash('0xabcd'), h.bytes(100), 0, '0xabcd')
    // dispute value
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    // vote
    await gov.connect(user1).vote(1, true, false)
    await gov.connect(user2).vote(1, false, false)
    await gov.connect(reporter1).vote(1, true, false)
    await gov.connect(reporter2).vote(1, false, false)
    await gov.connect(tokenholder1).vote(1, true, false)
    await gov.connect(tokenholder2).vote(1, false, false)
    await gov.connect(multisig).vote(1, true, false)
    // tally and execute
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    await h.advanceTime(86400)
    await gov.executeVote(1)
    // checks
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[1][5] == web3.utils.toWei("959"), "Tokenholders doesSupport should be correct")
    assert(voteInfo[1][6] == web3.utils.toWei("30"), "Tokenholders against should be correct")
    assert(voteInfo[1][7] == 0, "Tokenholders invalid should be correct")
    assert(voteInfo[1][8] == web3.utils.toWei("1"), "Users doesSupport should be correct")
    assert(voteInfo[1][9] == web3.utils.toWei("1"), "Users against should be correct")
    assert(voteInfo[1][10] == 0, "Users invalid should be correct")
    assert(voteInfo[1][11] == 1, "Reporters doesSupport should be correct")
    assert(voteInfo[1][12] == 1, "Reporters against should be correct")
    assert(voteInfo[1][13] == 0, "Reporters invalid should be correct")
    assert(voteInfo[1][14] == 1, "Multisig doesSupport should be correct")
    assert(voteInfo[1][15] == 0, "Multisig against should be correct")
    assert(voteInfo[1][16] == 0, "Multisig invalid should be correct")
    assert(voteInfo[3] == 1, "Vote result should be correct")
  });
  it("Test query id value after a dispute", async () => {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("10"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    // there should be no previous value
    await h.expectThrow(flex.getCurrentValue(ETH_QUERY_ID))
    // submit first value
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(200), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    assert(await flex.getCurrentValue(ETH_QUERY_ID) == h.bytes(200), "Current value should be most recent report")
    // bypass reporter lock
    h.advanceTime(43200)
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    // 'Dispute must be started within reporting lock time'
    await h.expectThrow(gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    assert(await flex.getCurrentValue(ETH_QUERY_ID) == h.bytes(100), "Current value should be most recent report")
    blocky = await h.getBlock()
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await flex.getCurrentValue(ETH_QUERY_ID) == h.bytes(200), "Current value shouldn't be dispute value");
  })
  it("Cannot vote on dispute id 0", async function() {
    await h.expectThrow(gov.vote(0, true, false))
    await token.connect(accounts[1]).approve(flex.address, h.toWei("10"))
    await flex.connect(accounts[1]).depositStake(h.toWei("10"))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).transfer(accounts[2].address, h.toWei("100"))
    await token.connect(accounts[2]).approve(gov.address, h.toWei("10"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await h.expectThrow(gov.connect(accounts[2]).vote(0, true, false))
    await gov.connect(accounts[2]).vote(1, true, false)
  })

  it("On multiple vote rounds, disputed value gets recorded correctly", async function() {
    await token.connect(accounts[1]).approve(flex.address, h.toWei("10"))
    await flex.connect(accounts[1]).depositStake(h.toWei("10"))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).transfer(accounts[2].address, h.toWei("100"))
    await token.connect(accounts[2]).approve(gov.address, h.toWei("100"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    disputeInfo = await gov.getDisputeInfo(1)
    assert(disputeInfo[2] == h.bytes(100), "Disputed value should be correct")
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    disputeInfo = await gov.getDisputeInfo(2)
    assert(disputeInfo[2] == h.bytes(100), "Disputed value should be correct")
  })

  it("Test array of dispute ids by reporter address", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, h.toWei("20"))
    await token.connect(accounts[1]).transfer(accounts[3].address, h.toWei("50"))
    await token.connect(accounts[1]).approve(flex.address, h.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, h.toWei("20"))
    await token.connect(accounts[3]).approve(gov.address, h.toWei("50"))
    await flex.connect(accounts[1]).depositStake(h.toWei("20"))
    await flex.connect(accounts[2]).depositStake(h.toWei("20"))
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky1 = await h.getBlock()
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.bytes(200), 0, ETH_QUERY_DATA)
    blocky2 = await h.getBlock()
    await gov.connect(accounts[3]).beginDispute(ETH_QUERY_ID, blocky1.timestamp)

    disputeIds = await gov.getDisputesByReporter(accounts[1].address)
    assert(disputeIds.length == 1, "Should be one dispute id")
    assert(disputeIds[0] == 1, "Dispute id should be correct")
    disputeIds = await gov.getDisputesByReporter(accounts[2].address)
    assert(disputeIds.length == 0, "Should be zero dispute id")
    await gov.connect(accounts[3]).beginDispute(ETH_QUERY_ID, blocky2.timestamp)
    disputeIds = await gov.getDisputesByReporter(accounts[2].address)
    assert(disputeIds.length == 1, "Should be one dispute id")
    assert(disputeIds[0] == 2, "Dispute id should be correct")
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    await gov.connect(accounts[3]).beginDispute(ETH_QUERY_ID, blocky1.timestamp)
    disputeIds = await gov.getDisputesByReporter(accounts[1].address)
    assert(disputeIds.length == 2, "Should be two dispute ids")
    assert(disputeIds[0] == 1, "Dispute id should be correct")
    assert(disputeIds[1] == 3, "Dispute id should be correct")
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(300), 0, ETH_QUERY_DATA)
    blocky3 = await h.getBlock()
    await gov.connect(accounts[3]).beginDispute(ETH_QUERY_ID, blocky3.timestamp)
    disputeIds = await gov.getDisputesByReporter(accounts[2].address)
    assert(disputeIds.length == 2, "Should be 2 dispute ids")
    assert(disputeIds[0] == 2, "Dispute id should be correct")
    assert(disputeIds[1] == 4, "Dispute id should be correct")
  })

  it("test vote weight outcomes", async function() {
    // setup
    await token.connect(accounts[1]).transfer(accounts[0].address, h.toWei("1000"))
    await token.transfer(accounts[10].address, h.toWei("40"))
    await token.transfer(accounts[1].address, h.toWei("1"))
    await token.transfer(accounts[2].address, h.toWei("2"))
    await token.transfer(accounts[3].address, h.toWei("3"))
    await token.transfer(accounts[4].address, h.toWei("1"))
    await token.approve(gov.address, h.toWei("40"))
    await token.connect(accounts[10]).approve(flex.address, h.toWei("40"))
    await flex.connect(accounts[10]).depositStake(h.toWei("40"))

    // support > against + invalid
    await flex.connect(accounts[10]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await gov.beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[3]).vote(1, true, false) // support - 3
    await gov.connect(accounts[4]).vote(1, false, false) // against - 1
    await gov.connect(accounts[1]).vote(1, false, true) // invalid - 1
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(1)
    // get vote outcome
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[3] == 1, "Result should be support")

    // against > support + invalid
    await flex.connect(accounts[10]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await gov.beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[4]).vote(2, true, false) // support - 1
    await gov.connect(accounts[3]).vote(2, false, true) // against - 3
    await gov.connect(accounts[1]).vote(2, false, true) // invalid - 1
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(2)
    // get vote outcome
    voteInfo = await gov.getVoteInfo(3)
    assert(voteInfo[3] == 0, "Result should be against")

    // support <= against + invalid; against <= support + invalid; support > invalid; support > against
    await flex.connect(accounts[10]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await gov.beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[2]).vote(3, true, false) // support - 2
    await gov.connect(accounts[4]).vote(3, false, false) // against - 1
    await gov.connect(accounts[1]).vote(3, false, true) // invalid - 1
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(3)
    // get vote outcome
    voteInfo = await gov.getVoteInfo(3)
    assert(voteInfo[3] == 2, "Result should be invalid")

    // support <= against + invalid; against <= support + invalid; against > invalid; against > support
    await flex.connect(accounts[10]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await gov.beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[4]).vote(4, true, false) // support - 1
    await gov.connect(accounts[2]).vote(4, false, true) // against - 2
    await gov.connect(accounts[1]).vote(4, false, true) // invalid - 1
    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(4)
    // get vote outcome
    voteInfo = await gov.getVoteInfo(4)
    assert(voteInfo[3] == 2, "Result should be invalid")
  })

  it("Dispute fee capped at stake amount, one report", async function() {
    reporter1 = accounts[9]
    disputer = accounts[5]
    await token.mint(reporter1.address, h.toWei("1000"))
    await token.mint(disputer.address, h.toWei("100"))

    await token.connect(reporter1).approve(flex.address, h.toWei("1000"))
    await flex.connect(reporter1).depositStake(h.toWei("1000"))
    await flex.connect(reporter1).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    assert(await flex.getStakeAmount() == h.toWei("10"), "Stake amount should be correct")
    assert(await token.balanceOf(disputer.address) == h.toWei("100"), "Disputer should have correct balance")
    await token.connect(disputer).approve(gov.address, h.toWei("1"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("99"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[1][3] == h.toWei("1"), "Dispute fee should be correct")

    await h.advanceTime(86400 * 1)
    await gov.tallyVotes(1)
    await token.connect(disputer).approve(gov.address, h.toWei("2"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("97"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(2)
    assert(voteInfo[1][3] == h.toWei("2"), "Dispute fee should be correct")

    await h.advanceTime(86400 * 2)
    await gov.tallyVotes(2)
    await token.connect(disputer).approve(gov.address, h.toWei("4"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("93"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(3)
    assert(voteInfo[1][3] == h.toWei("4"), "Dispute fee should be correct")

    await h.advanceTime(86400 * 3)
    await gov.tallyVotes(3)
    await token.connect(disputer).approve(gov.address, h.toWei("8"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("85"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(4)
    assert(voteInfo[1][3] == h.toWei("8"), "Dispute fee should be correct")

    await h.advanceTime(86400 * 4)
    await gov.tallyVotes(4)
    await token.connect(disputer).approve(gov.address, h.toWei("10"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("75"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(5)
    assert(voteInfo[1][3] == h.toWei("10"), "Dispute fee should be correct")

    await h.advanceTime(86400 * 5)
    await gov.tallyVotes(5)
    await token.connect(disputer).approve(gov.address, h.toWei("10"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("65"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(6)
    assert(voteInfo[1][3] == h.toWei("10"), "Dispute fee should be correct")
  })

  it("Dispute fee capped at stake amount, many reports", async function() {
    reporter1 = accounts[9]
    reporter2 = accounts[10]
    reporter3 = accounts[11]
    reporter4 = accounts[12]
    reporter5 = accounts[13]
    reporter6 = accounts[14]
    disputer = accounts[5]
    await token.mint(reporter1.address, h.toWei("1000"))
    await token.mint(reporter2.address, h.toWei("1000"))
    await token.mint(reporter3.address, h.toWei("1000"))
    await token.mint(reporter4.address, h.toWei("1000"))
    await token.mint(reporter5.address, h.toWei("1000"))
    await token.mint(reporter6.address, h.toWei("1000"))
    await token.mint(disputer.address, h.toWei("100"))

    await token.connect(reporter1).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter2).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter3).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter4).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter5).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter6).approve(flex.address, h.toWei("1000"))

    await flex.connect(reporter1).depositStake(h.toWei("1000"))
    await flex.connect(reporter2).depositStake(h.toWei("1000"))
    await flex.connect(reporter3).depositStake(h.toWei("1000"))
    await flex.connect(reporter4).depositStake(h.toWei("1000"))
    await flex.connect(reporter5).depositStake(h.toWei("1000"))
    await flex.connect(reporter6).depositStake(h.toWei("1000"))

    await flex.connect(reporter1).submitValue(ETH_QUERY_ID, h.bytes(101), 0, ETH_QUERY_DATA)
    blocky1 = await h.getBlock()
    await flex.connect(reporter2).submitValue(ETH_QUERY_ID, h.bytes(102), 0, ETH_QUERY_DATA)
    blocky2 = await h.getBlock()
    await flex.connect(reporter3).submitValue(ETH_QUERY_ID, h.bytes(103), 0, ETH_QUERY_DATA)
    blocky3 = await h.getBlock()
    await flex.connect(reporter4).submitValue(ETH_QUERY_ID, h.bytes(104), 0, ETH_QUERY_DATA)
    blocky4 = await h.getBlock()
    await flex.connect(reporter5).submitValue(ETH_QUERY_ID, h.bytes(105), 0, ETH_QUERY_DATA)
    blocky5 = await h.getBlock()
    await flex.connect(reporter6).submitValue(ETH_QUERY_ID, h.bytes(106), 0, ETH_QUERY_DATA)
    blocky6 = await h.getBlock()

    assert(await flex.getStakeAmount() == h.toWei("10"), "Stake amount should be correct")
    assert(await token.balanceOf(disputer.address) == h.toWei("100"), "Disputer should have correct balance")
    await token.connect(disputer).approve(gov.address, h.toWei("1"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky1.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("99"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[1][3] == h.toWei("1"), "Dispute fee should be correct")

    await token.connect(disputer).approve(gov.address, h.toWei("2"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky2.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("97"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(2)
    assert(voteInfo[1][3] == h.toWei("2"), "Dispute fee should be correct")

    await token.connect(disputer).approve(gov.address, h.toWei("4"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky3.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("93"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(3)
    assert(voteInfo[1][3] == h.toWei("4"), "Dispute fee should be correct")

    await token.connect(disputer).approve(gov.address, h.toWei("8"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky4.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("85"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(4)
    assert(voteInfo[1][3] == h.toWei("8"), "Dispute fee should be correct")

    await token.connect(disputer).approve(gov.address, h.toWei("10"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky5.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("75"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(5)
    assert(voteInfo[1][3] == h.toWei("10"), "Dispute fee should be correct")

    await token.connect(disputer).approve(gov.address, h.toWei("10"))
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky6.timestamp)
    assert(await token.balanceOf(disputer.address) == h.toWei("65"), "Disputer should have correct balance")
    voteInfo = await gov.getVoteInfo(6)
    assert(voteInfo[1][3] == h.toWei("10"), "Dispute fee should be correct")
  })
})
