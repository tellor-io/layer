const {expect,assert} = require("chai");
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

describe("Governance Function Tests", function() {

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
    gov = await Governance.deploy(flex.address, accounts[0].address);
    await gov.deployed();
    await flex.init(gov.address)
    const Autopay = await ethers.getContractFactory("AutopayMock");
    autopay = await Autopay.deploy(token.address);
    await token.mint(accounts[1].address, web3.utils.toWei("1000"));
    autopayArray = abiCoder.encode(["address[]"], [[autopay.address]]);
  });
  it("Test Constructor()", async function() {
    assert(await gov.tellor() == flex.address, "tellor address should be correct")
    assert(await gov.oracle() == flex.address, "oracle address should be set")
    assert(await gov.token() == token.address, "token address should be set")
    assert(await gov.getDisputeFee() == await flex.getStakeAmount()/10, "min dispute fee should be set properly")
    assert(await gov.teamMultisig() == accounts[0].address, "team multisig should be set correctly")
  });
  it("Test beginDispute()", async function() {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("1000"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    await token.connect(accounts[1]).transfer(accounts[3].address, web3.utils.toWei("100"))
    let blocky = await h.getBlock()
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    let balance1 = await token.balanceOf(accounts[2].address)
    await h.expectThrow(gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) //no value exists for the timestamp provided
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await h.expectThrow(gov.connect(accounts[4]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) // must have tokens to pay/begin dispute
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp);
    let balance2 = await token.balanceOf(accounts[2].address)
    let vars = await gov.getDisputeInfo(1)
    let vars2 = await gov.getVoteInfo(1)
    let _hash = ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [ETH_QUERY_ID, blocky.timestamp])
    assert(await gov.getVoteCount() == 1, "vote count should be correct")
    assert(vars[0] == ETH_QUERY_ID, "queryID should be correct")
    assert(vars[1] == blocky.timestamp, "timestamp should be correct")
    assert(vars[2] == h.bytes(100), "value should be correct")
    assert(vars[3] == accounts[1].address, "accounts[1] should be correct")
    assert(vars2[4] == accounts[2].address, "initiator should be correct")
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 1, "open disputes on ID should be correct")
    assert(await gov.getVoteRounds(_hash) == 1, "number of vote rounds should be correct")
    assert(balance1 - balance2 - (await flex.getStakeAmount()/10) == 0, "dispute fee paid should be correct")
    await h.advanceTime(86400 * 2);
    await gov.tallyVotes(1)
    await h.advanceTime(86400 * 2);
    await gov.executeVote(1)
    await h.advanceTime(86400 * 2)
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await h.expectThrow(gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) //assert second dispute started within a day
    await token.connect(accounts[3]).approve(flex.address, web3.utils.toWei("1000"))
    await flex.connect(accounts[3]).depositStake(web3.utils.toWei("10"))
    await flex.connect(accounts[3]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await h.advanceTime(86400 + 10)
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await h.expectThrow(gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) //dispute must be started within timeframe
  });
  it("Test executeVote()", async function() {
    await token.connect(accounts[1]).approve(flex.address, web3.utils.toWei("1000"))
    await flex.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("100"))
    await token.connect(accounts[1]).transfer(accounts[3].address, web3.utils.toWei("100"))
    let blocky = await h.getBlock()
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    let balance1 = await token.balanceOf(accounts[2].address)
    await h.expectThrow(gov.connect(accounts[2]).executeVote(1)) //1 vote ID must be valid
    await flex.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await h.expectThrow(gov.connect(accounts[4]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) // must have tokens to pay for dispute
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp);
    let balance2 = await token.balanceOf(accounts[2].address)
    let vars = await gov.getDisputeInfo(1)
    let _hash = ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [ETH_QUERY_ID, blocky.timestamp])
    assert(vars[0] == ETH_QUERY_ID, "queryID should be correct")
    assert(vars[1] == blocky.timestamp, "timestamp should be correct")
    assert(vars[2] == h.bytes(100), "value should be correct")
    assert(vars[3] == accounts[1].address, "accounts[1] should be correct")
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 1, "open disputes on ID should be correct")
    assert(await gov.getVoteRounds(_hash) == 1, "number of vote rounds should be correct")
    assert(balance1 - balance2 - web3.utils.toWei("1") == 0, "dispute fee paid should be correct")
    await h.expectThrow(gov.connect(accounts[2]).executeVote(10)) //dispute id must exist
    await h.advanceTime(86400 * 2);
    await h.expectThrow(gov.connect(accounts[2]).executeVote(1)) //vote must be tallied
    await gov.connect(accounts[2]).vote(1, true, false)
    await gov.tallyVotes(1)
    await h.expectThrow(gov.connect(accounts[2]).executeVote(1)) //a day must pass before execution
    await h.advanceTime(86400 * 2);
    await gov.executeVote(1)
    await h.expectThrow(gov.connect(accounts[2]).executeVote(1)) //vote already executed
    await h.advanceTime(86400 * 2)
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await h.expectThrow(gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)) //assert second dispute started within a day
    vars = await gov.getVoteInfo(1)
    assert(vars[0] == _hash, "hash should be correct")
    assert(vars[1][0] == 1, "vote round should be correct")
    assert(vars[2] == true, "vote should be executed")
    assert(vars[3] == true, "vote should pass")
    await token.connect(accounts[3]).approve(flex.address, web3.utils.toWei("1000"))
    await flex.connect(accounts[3]).depositStake(web3.utils.toWei("10"))
    await flex.connect(accounts[3]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp) //fast forward assert
    await h.advanceTime(86400 * 2);
    await gov.tallyVotes(2)
    await token.connect(accounts[2]).approve(gov.address, web3.utils.toWei("20"))
    await gov.connect(accounts[2]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await h.advanceTime(86400 * 2);
    await h.expectThrow(gov.connect(accounts[2]).executeVote(1)) //vote must be the final vote
    await h.advanceTime(86400 * 2);
    await gov.tallyVotes(3)
    await h.expectThrow(gov.connect(accounts[2]).executeVote(3)) //must wait longer 
    await h.advanceTime(86400);
    await gov.connect(accounts[2]).executeVote(3)
  });
  it("Test tallyVotes()", async function() {
    // Test tallyVotes on dispute
    // tallyVotes (1dispute could not have been executed, 2)or tallied, 
    // 3)dispute does not exist 4) cannot tally before the voting time has ended)
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await h.expectThrow(gov.connect(accounts[1]).tallyVotes(1)) // Cannot tally a dispute that does not exist
    
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(accounts[1]).vote(1, true, false)
    await h.expectThrow(gov.connect(accounts[1]).tallyVotes(1)) // Time for voting has not elapsed
    await h.advanceTime(86400 * 2)
    await gov.connect(accounts[1]).tallyVotes(1)
    blocky = await h.getBlock()
    await h.advanceTime(86400)
    await h.expectThrow(gov.connect(accounts[1]).tallyVotes(1)) // cannot re-tally a dispute --BL
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[3] == 1, "Vote result should change")
    assert(voteInfo[1][4] == blocky.timestamp, "Tally date should be correct")
    await gov.executeVote(1)
    await h.expectThrow(gov.connect(accounts[1]).tallyVotes(1)) // Dispute has been already executed
    assert(await token.balanceOf(accounts[2].address)== 0, "should not have tokens returned")
  });
  it("Test vote()", async function() {
    // vote (1 dispute must exist, 2)cannot have been tallied, 3)sender has already voted)
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await h.expectThrow(gov.connect(accounts[1]).vote(2, true, false)) // Can't vote on dispute does not exist
    await gov.connect(accounts[1]).vote(1, true, false)
    await gov.connect(accounts[2]).vote(1, false, false)
    await h.expectThrow(gov.connect(accounts[1]).vote(1, true, false)) // Sender has already voted
    await h.advanceTime(86400 * 2)
    await gov.connect(accounts[1]).tallyVotes(1)
    await h.expectThrow(gov.connect(accounts[1]).vote(1, true, false)) // Vote has already been tallied
    voteInfo = await gov.getVoteInfo(1)
    assert(voteInfo[1][5] - await token.balanceOf(accounts[1].address) == 0, "Tokenholders doesSupport tally should be correct")
    assert(voteInfo[1][6] == web3.utils.toWei("10"), "Tokenholders against tally should be correct")
    assert(voteInfo[1][7] == 0, "Tokenholders invalid tally should be correct")
    assert(voteInfo[1][8] == 0, "Users doesSupport tally should be correct")
    assert(voteInfo[1][9] == 0, "Users against tally should be correct")
    assert(voteInfo[1][10] == 0, "Users invalid tally should be correct")
    assert(voteInfo[1][11] == 0, "Reporters doesSupport tally should be correct")
    assert(voteInfo[1][12] == 1, "Reporters against tally should be correct")
    assert(voteInfo[1][13] == 0, "Reporters invalid tally should be correct")
    assert(voteInfo[1][14] == 0, "teamMultisig doesSupport tally should be correct")
    assert(voteInfo[1][15] == 0, "teamMultisig against tally should be correct")
    assert(voteInfo[1][16] == 0, "teamMultisig invalid tally should be correct")
    assert(await gov.didVote(1, accounts[1].address), "Voter's voted status should be correct")
    assert(await gov.didVote(1, accounts[2].address), "Voter's voted status should be correct")
    assert(await gov.didVote(1, accounts[3].address) == false, "Voter's voted status should be correct")
    assert(await gov.getVoteTallyByAddress(accounts[1].address) == 1, "Vote tally by address should be correct")
    assert(await gov.getVoteTallyByAddress(accounts[2].address) == 1, "Vote tally by address should be correct")
  });
  it("Test voteOnMultipleDisputes()", async function() {
    reporter1 = accounts[9]
    reporter2 = accounts[10]
    reporter3 = accounts[11]
    disputer = accounts[5]
    voter = accounts[6]

    await token.mint(reporter1.address, h.toWei("1000"))
    await token.mint(reporter2.address, h.toWei("1000"))
    await token.mint(reporter3.address, h.toWei("1000"))
    await token.mint(disputer.address, h.toWei("100"))
    await token.mint(voter.address, h.toWei("100"))

    await token.connect(reporter1).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter2).approve(flex.address, h.toWei("1000"))
    await token.connect(reporter3).approve(flex.address, h.toWei("1000"))
    await token.connect(disputer).approve(gov.address, h.toWei("100"))

    await flex.connect(reporter1).depositStake(h.toWei("1000"))
    await flex.connect(reporter2).depositStake(h.toWei("1000"))
    await flex.connect(reporter3).depositStake(h.toWei("1000"))

    await flex.connect(reporter1).submitValue(ETH_QUERY_ID, h.bytes(101), 0, ETH_QUERY_DATA)
    blocky1 = await h.getBlock()
    await flex.connect(reporter2).submitValue(ETH_QUERY_ID, h.bytes(102), 0, ETH_QUERY_DATA)
    blocky2 = await h.getBlock()
    await flex.connect(reporter3).submitValue(ETH_QUERY_ID, h.bytes(103), 0, ETH_QUERY_DATA)
    blocky3 = await h.getBlock()

    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky1.timestamp)
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky2.timestamp)
    await gov.connect(disputer).beginDispute(ETH_QUERY_ID, blocky3.timestamp)

    await gov.connect(voter).voteOnMultipleDisputes([1, 2, 3], [true, false, false], [false, false, true])

    voteInfo1 = await gov.getVoteInfo(1)
    voteInfo2 = await gov.getVoteInfo(2)
    voteInfo3 = await gov.getVoteInfo(3)

    assert(voteInfo1[1][5] == h.toWei("100"), "Dispute1 Tokenholders doesSupport tally should be correct")
    assert(voteInfo2[1][6] == h.toWei("100"), "Dispute2 Tokenholders against tally should be correct")
    assert(voteInfo3[1][7] == h.toWei("100"), "Dispute3 Tokenholders invalid tally should be correct")
  })
  it("Test didVote()", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await gov.didVote(1, accounts[1].address) == false, "Voter's voted status should be correct")
    await gov.connect(accounts[1]).vote(1, true, false)
    assert(await gov.didVote(1, accounts[1].address), "Voter's voted status should be correct")
  });
  it("Test getDisputeInfo()", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    disputeInfo = await gov.getDisputeInfo(1)
    assert(disputeInfo[0] == ETH_QUERY_ID, "Disputed query id should be correct")
    assert(disputeInfo[1] == blocky.timestamp, "Disputed timestamp should be correct")
    assert(disputeInfo[2] == h.uintTob32(100), "Disputed value should be correct")
    assert(disputeInfo[3] == accounts[2].address, "Disputed reporter should be correct")
  });
  it("Test getOpenDisputesOnId()", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).transfer(accounts[3].address, web3.utils.toWei("20"))
    await token.connect(accounts[3]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[3]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[3]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    let blocky2 = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 0, "Open disputes on ID should be correct")
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 1, "Open disputes on ID should be correct")
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky2.timestamp)
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 2, "Open disputes on ID should be correct")
    await gov.connect(accounts[1]).vote(1, true, false)
    await h.advanceTime(86400 * 2)
    await gov.connect(accounts[1]).tallyVotes(1)
    await h.advanceTime(86400)
    await gov.executeVote(1)
    assert(await gov.getOpenDisputesOnId(ETH_QUERY_ID) == 1, "Open disputes on ID should be correct")
  });
  it("Test getVoteCount()", async function() {
    assert(await gov.getVoteCount() == 0, "Vote count should start at 0")
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await gov.getVoteCount() == 1, "Vote count should increment correctly")
    await h.advanceTime(86400 * 2)
    await gov.connect(accounts[1]).tallyVotes(1)
    await h.advanceTime(86400)
    await gov.executeVote(1)
    assert(await gov.getVoteCount() == 1, "Vote count should not change after vote execution")
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    assert(await gov.getVoteCount() == 2, "Vote count should increment correctly")
  });
  it("Test getVoteInfo()", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky0 = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky0.timestamp)
    blocky1 = await h.getBlock()
    await gov.connect(accounts[1]).vote(1, true, false)
    await h.advanceTime(86400 * 7)
    await gov.connect(accounts[1]).tallyVotes(1)
    blocky2 = await h.getBlock()
    await h.advanceTime(86400)
    await gov.executeVote(1)
    voteInfo = await gov.getVoteInfo(1)
    hash = ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [ETH_QUERY_ID, blocky0.timestamp])
    assert(voteInfo[0] == hash, "Vote hash should be correct")
    assert(voteInfo[1][0] == 1, "Vote round should be correct")
    assert(voteInfo[1][1] == blocky1.timestamp, "Vote start date should be correct")
    assert(voteInfo[1][2] == blocky1.number, "Vote blocknumber should be correct")
    assert(voteInfo[1][3] - web3.utils.toWei("1") == 0, "Vote fee should be correct")
    assert(voteInfo[1][4] == blocky2.timestamp, "Vote tallyDate should be correct")
    assert(voteInfo[1][5] == web3.utils.toWei("979"), "Vote tokenholders doesSupport should be correct")
    assert(voteInfo[1][6] == 0, "Vote tokenholders against should be correct")
    assert(voteInfo[1][7] == 0, "Vote tokenholders invalid should be correct")
    assert(voteInfo[1][8] == 0, "Vote users doesSupport should be correct")
    assert(voteInfo[1][9] == 0, "Vote users against should be correct")
    assert(voteInfo[1][10] == 0, "Vote users invalid should be correct")
    assert(voteInfo[1][11] == 0, "Vote reporters doesSupport should be correct")
    assert(voteInfo[1][12] == 0, "Vote reporters against should be correct")
    assert(voteInfo[1][13] == 0, "Vote reporters invalid should be correct")
    assert(voteInfo[1][14] == 0, "Vote teamMultisig doesSupport should be correct")
    assert(voteInfo[1][15] == 0, "Vote teamMultisig against should be correct")
    assert(voteInfo[1][16] == 0, "Vote teamMultisig invalid should be correct")
    assert(voteInfo[2] == true, "Vote executed should be true")
    assert(voteInfo[3] == 1, "Vote result should be PASSED")
    assert(voteInfo[4] == accounts[1].address, "Vote initiator address should be correct")
  });
  it("Test getVoteRounds()", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky0 = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky0.timestamp)
    blocky1 = await h.getBlock()
    hash = ethers.utils.solidityKeccak256(['bytes32', 'uint256'], [ETH_QUERY_ID, blocky0.timestamp])
    voteRounds = await gov.getVoteRounds(hash)
    assert(voteRounds.length == 1, "Vote rounds length should be correct")
    assert(voteRounds[0] == 1, "Vote rounds disputeIds should be correct")
    await h.advanceTime(86400 * 2)
    await gov.connect(accounts[1]).tallyVotes(1)
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("20"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky0.timestamp)
    voteRounds = await gov.getVoteRounds(hash)
    assert(voteRounds.length == 2, "Vote rounds length should be correct")
    assert(voteRounds[0] == 1, "Vote round disputeId should be correct")
    assert(voteRounds[1] == 2, "Vote round disputeId should be correct")
  });
  it("Test getVoteTallyByAddress()", async function() {
    await token.connect(accounts[1]).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky0 = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky0.timestamp)
    await token.connect(accounts[1]).transfer(accounts[3].address, web3.utils.toWei("20"))
    await token.connect(accounts[3]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[3]).depositStake(web3.utils.toWei("20"))
    await flex.connect(accounts[3]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky0 = await h.getBlock()
    await token.connect(accounts[1]).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(accounts[1]).beginDispute(ETH_QUERY_ID, blocky0.timestamp)
    assert(await gov.getVoteTallyByAddress(accounts[1].address) == 0, "Vote tally should be correct")
    await gov.connect(accounts[1]).vote(1, true, false)
    assert(await gov.getVoteTallyByAddress(accounts[1].address) == 1, "Vote tally should be correct")
    await gov.connect(accounts[1]).vote(2, true, false)
    assert(await gov.getVoteTallyByAddress(accounts[1].address) == 2, "Vote tally should be correct")
  })
  it("Test _getUserTips", async function() {
    let userAccount = accounts[1]
    await token.connect(userAccount).transfer(accounts[2].address, web3.utils.toWei("20"))
    await token.connect(accounts[2]).approve(flex.address, web3.utils.toWei("20"))
    await flex.connect(accounts[2]).depositStake(web3.utils.toWei("20"))
    // submit autopay address[] to oracle
    await flex.connect(accounts[2]).submitValue(autopayQueryId, autopayArray, 0, autopayQueryData)
    h.advanceTime(43200)
    await token.connect(userAccount).approve(autopay.address, web3.utils.toWei("20"))
    await autopay.connect(userAccount).tip(ETH_QUERY_ID,web3.utils.toWei("20"),h.bytes(100))
    await flex.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(100), 0, ETH_QUERY_DATA)
    blocky = await h.getBlock()
    await token.connect(userAccount).approve(gov.address, web3.utils.toWei("10"))
    await gov.connect(userAccount).beginDispute(ETH_QUERY_ID, blocky.timestamp)
    await gov.connect(userAccount).vote(1, true, false)
    await h.advanceTime(86400 * 7)
    await gov.connect(userAccount).tallyVotes(1)
    await h.advanceTime(86400)
    await gov.executeVote(1)
    voteInfo = await gov.getVoteInfo(1)
    // user weight based on tip amount
    assert(voteInfo[1][8] == web3.utils.toWei("20"), "Vote users doesSupport weight should be based on tip total")
  })
});