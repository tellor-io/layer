const { expect } = require("chai");
const { network, ethers } = require("hardhat");
const h = require("./helpers/helpers");
const web3 = require('web3');
const BN = ethers.BigNumber.from
var assert = require('assert');

describe("TellorFlex - Function Tests", function () {

	let tellor;
	let token;
	let governance;
	let govSigner;
	let accounts;
	let owner;
    const MINIMUM_STAKE_AMOUNT = web3.utils.toWei("100")
	const STAKE_AMOUNT_USD_TARGET = web3.utils.toWei("500");
	const PRICE_TRB = web3.utils.toWei("50");
	const REPORTING_LOCK = 43200; // 12 hours
	const QUERYID1 = h.uintTob32(1)
	const QUERYID2 = h.uintTob32(2)
	const REWARD_RATE_TARGET = 60 * 60 * 24 * 30; // 30 days
	const abiCoder = new ethers.utils.AbiCoder
	const TRB_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["trb", "usd"])
	const TRB_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", TRB_QUERY_DATA_ARGS])
	const TRB_QUERY_ID = ethers.utils.keccak256(TRB_QUERY_DATA)
	const ETH_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["eth", "usd"])
    const ETH_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", ETH_QUERY_DATA_ARGS])
    const ETH_QUERY_ID = ethers.utils.keccak256(ETH_QUERY_DATA)
	const BTC_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["btc", "usd"])
	const BTC_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", BTC_QUERY_DATA_ARGS])
	const BTC_QUERY_ID = ethers.utils.keccak256(BTC_QUERY_DATA)
	const smap = {
        startDate: 0,
        stakedBalance: 1,
        lockedBalance: 2,
        rewardDebt: 3,
        reporterLastTimestamp: 4,
        reportsSubmitted: 5,
        startVoteCount: 6,
        startVoteTally: 7,
        staked: 8
    } // getStakerInfo() indices

	beforeEach(async function () {
		accounts = await ethers.getSigners();
		owner = accounts[0]
		const ERC20 = await ethers.getContractFactory("StakingToken");
		token = await ERC20.deploy();
		await token.deployed();
		const Governance = await ethers.getContractFactory("GovernanceMock");
		governance = await Governance.deploy();
		await governance.deployed();
		const TellorFlex = await ethers.getContractFactory("TestFlex");
		tellor = await TellorFlex.deploy(token.address, REPORTING_LOCK, STAKE_AMOUNT_USD_TARGET, PRICE_TRB, MINIMUM_STAKE_AMOUNT, TRB_QUERY_ID);
		owner = await ethers.getSigner(await tellor.owner())
		await tellor.deployed();
		await governance.setTellorAddress(tellor.address);
		await token.mint(accounts[1].address, web3.utils.toWei("1000"));
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await hre.network.provider.request({
			method: "hardhat_impersonateAccount",
			params: [governance.address]
		}
		)

		govSigner = await ethers.getSigner(governance.address);
		await accounts[10].sendTransaction({ to: governance.address, value: ethers.utils.parseEther("1.0") });

		await tellor.connect(owner).init(governance.address)
	});

	it("constructor", async function () {
		let stakeAmount = await tellor.getStakeAmount()
		expect(stakeAmount).to.equal(MINIMUM_STAKE_AMOUNT);
		let governanceAddress = await tellor.getGovernanceAddress()
		expect(governanceAddress).to.equal(governance.address)
		// test require: token address must not be 0
		let tokenAddress = await tellor.getTokenAddress()
		expect(tokenAddress).to.equal(token.address)
		let reportingLock = await tellor.getReportingLock()
		expect(reportingLock).to.equal(REPORTING_LOCK)
	});

	it("depositStake", async function () {
		expect(await token.balanceOf(accounts[1].address)).to.equal(web3.utils.toWei("1000"))
		expect(await token.balanceOf(accounts[2].address)).to.equal(0)
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await token.connect(accounts[2]).approve(tellor.address, web3.utils.toWei("1000"))

		// test require(token.transferFrom... when locked balance <= zero
		await h.expectThrow(tellor.connect(accounts[2]).depositStake(web3.utils.toWei("100")))

		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("100"))
		let blocky = await h.getBlock()
		expect(await token.balanceOf(accounts[1].address)).to.equal(web3.utils.toWei("900"))
		expect(await tellor.getTotalStakers()).to.equal(1)
		let stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.startDate]).to.equal(blocky.timestamp) // startDate
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("100")) // stakedBalance
		expect(stakerDetails[smap.lockedBalance]).to.equal(0) // lockedBalance
		expect(stakerDetails[smap.rewardDebt]).to.equal(0) // rewardDebt
		expect(stakerDetails[smap.reporterLastTimestamp]).to.equal(0) // reporterLastTimestamp
		expect(stakerDetails[smap.reportsSubmitted]).to.equal(0) // reportsSubmitted
		expect(stakerDetails[smap.startVoteCount]).to.equal(0) // startVoteCount
		expect(stakerDetails[smap.startVoteTally]).to.equal(0) // startVoteTally
		expect(stakerDetails[smap.staked]).to.equal(true) // staked
		expect(await tellor.totalRewardDebt()).to.equal(0)
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("100"))

		// Test min value for _amount argument
		await tellor.connect(accounts[3]).depositStake(0)
		expect(await tellor.getTotalStakers()).to.equal(1)

		await tellor.connect(accounts[1]).requestStakingWithdraw(h.toWei("5"))
		// test require(token.transferFrom... when locked balance above zero
		await tellor.connect(accounts[1]).depositStake(h.toWei("10"))
		expect(await token.balanceOf(accounts[1].address)).to.equal(web3.utils.toWei("895"))
		expect(await tellor.getTotalStakers()).to.equal(1) // Ensure only unique addresses add to total stakers
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("105"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("105"))
	})

	it("removeValue", async function () {
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await tellor.connect(accounts[1]).depositStake(MINIMUM_STAKE_AMOUNT)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
		let blocky = await h.getBlock()

		expect(await tellor.getNewValueCountbyQueryId(ETH_QUERY_ID)).to.equal(1)
		await h.expectThrow(tellor.connect(govSigner).removeValue(ETH_QUERY_ID, 500)) // invalid value
		expect(await tellor.retrieveData(ETH_QUERY_ID, blocky.timestamp)).to.equal(h.bytes(100))
		await h.expectThrow(tellor.connect(accounts[1]).removeValue(ETH_QUERY_ID, blocky.timestamp)) // test require: only gov can removeValue
		expect(await tellor.isInDispute(ETH_QUERY_ID, blocky.timestamp)).to.be.false
		await tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky.timestamp)
		expect(await tellor.getNewValueCountbyQueryId(ETH_QUERY_ID)).to.equal(1)
		expect(await tellor.retrieveData(ETH_QUERY_ID, blocky.timestamp)).to.equal("0x")
		expect(await tellor.isInDispute(ETH_QUERY_ID, blocky.timestamp)).to.be.true
		await h.expectThrow(tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky.timestamp)) // test require: value already disputed

		// Test min/max values for _timestamp argument
		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(100), 0, BTC_QUERY_DATA)
		await expect(tellor.connect(govSigner).removeValue(BTC_QUERY_ID, 0)).to.be.revertedWith("invalid timestamp")
		await expect(tellor.connect(govSigner).removeValue(BTC_QUERY_ID, ethers.constants.MaxUint256)).to.be.revertedWith("invalid timestamp")
	})

	it("requestStakingWithdraw", async function () {
		await h.expectThrow(tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("10"))) // test require: can't request staking withdraw when not staked

		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("1000"))
		let blocky = await h.getBlock()
		let stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.startDate]).to.equal(blocky.timestamp)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("1000"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(stakerDetails[smap.staked]).to.equal(true)
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("1000"))
		expect(await tellor.totalRewardDebt()).to.equal(0)
		await h.expectThrow(tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("1001"))) // test require: insufficient staked balance

		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("10"))
		blocky = await h.getBlock()
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.startDate]).to.equal(blocky.timestamp)
		expect(stakerDetails[smap.rewardDebt]).to.equal(0)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("990"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(web3.utils.toWei("10"))
		expect(stakerDetails[smap.staked]).to.equal(true)
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("990"))
		expect(await tellor.totalRewardDebt()).to.equal(0)

		// Test max/min for _amount arg
		await expect(tellor.connect(accounts[1]).requestStakingWithdraw(ethers.constants.MaxUint256)).to.be.revertedWith("insufficient staked balance")
		await tellor.connect(accounts[1]).requestStakingWithdraw(0)
		blocky = await h.getBlock()
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.startDate]).to.equal(blocky.timestamp)
		expect(stakerDetails[smap.rewardDebt]).to.equal(0)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("990"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(web3.utils.toWei("10"))
		expect(stakerDetails[smap.staked]).to.equal(true)
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("990"))
		expect(await tellor.totalRewardDebt()).to.equal(0)

		expect(await tellor.totalStakers()).to.equal(1)
		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("900"))
		expect(await tellor.totalStakers()).to.equal(0)
	})

	it("slashReporter", async function () {
		await h.expectThrow(tellor.connect(accounts[2]).slashReporter(accounts[1].address, accounts[2].address)) // test require: only gov can slash reporter
		await h.expectThrow(tellor.connect(govSigner).slashReporter(accounts[1].address, accounts[2].address)) // test require: can't slash non-staked address

		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("1000"))

		// Slash when lockedBalance = 0
		let stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("1000"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(await token.balanceOf(accounts[2].address)).to.equal(0)
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("1000"))
		await tellor.connect(govSigner).slashReporter(accounts[1].address, accounts[2].address)
		blocky0 = await h.getBlock()
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky0.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("900"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(stakerDetails[smap.staked]).to.equal(true)
		expect(await tellor.totalStakers()).to.equal(1) // Still one staker bc account#1 has 90 staked & stake amount is 10
		expect(await token.balanceOf(accounts[2].address)).to.equal(web3.utils.toWei("100"))
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("900"))

		// Slash when lockedBalance >= stakeAmount
		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("100"))
		blocky1 = await h.getBlock()
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("800"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(web3.utils.toWei("100"))
		await tellor.connect(govSigner).slashReporter(accounts[1].address, accounts[2].address)
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky1.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("800"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(await token.balanceOf(accounts[2].address)).to.equal(web3.utils.toWei("200"))
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("800"))

		// Slash when 0 < lockedBalance < stakeAmount
		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("5"))
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("795"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(web3.utils.toWei("5"))
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("795"))
		await tellor.connect(govSigner).slashReporter(accounts[1].address, accounts[2].address)
		blocky2 = await h.getBlock()
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky2.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("700"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(await token.balanceOf(accounts[2].address)).to.equal(web3.utils.toWei("300"))
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("700"))

		// Slash when lockedBalance + stakedBalance < stakeAmount
		// await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("625"))
		// await h.advanceTime(86400 * 7)
		// await tellor.connect(accounts[1]).withdrawStake()
		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("625"))
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("75"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(web3.utils.toWei("625"))
		expect(await tellor.totalStakeAmount()).to.equal(web3.utils.toWei("75"))
		await h.advanceTime(604800)
		await tellor.connect(accounts[1]).withdrawStake()
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(web3.utils.toWei("75"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(web3.utils.toWei("0"))
		await tellor.connect(govSigner).slashReporter(accounts[1].address, accounts[2].address)
		blocky = await h.getBlock()
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(0)
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		expect(await tellor.totalStakers()).to.equal(0)
		expect(await token.balanceOf(accounts[2].address)).to.equal(web3.utils.toWei("375"))
		expect(await tellor.totalStakeAmount()).to.equal(0)
	})

	it("submitValue", async function () {
		await token.mint(accounts[1].address, web3.utils.toWei("2000"))
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("10000"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("1200"))
		await h.expectThrow(tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID,'0x', 0, ETH_QUERY_DATA)) //must submit a value
		await h.expectThrow(tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.uintTob32(4000), 1, ETH_QUERY_DATA)) // test require: wrong nonce
		await h.expectThrow(tellor.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(4000), 1, ETH_QUERY_DATA)) // test require: insufficient staked balance
		await h.expectThrow(tellor.connect(accounts[1]).submitValue(h.uintTob32(101), h.uintTob32(4000), 0, ETH_QUERY_DATA)) // test require: non-legacy queryId must equal hash(queryData)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.expectThrow(tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.uintTob32(4000), 1, ETH_QUERY_DATA)) // test require: still in reporting lock

		await h.advanceTime(3600) // 1 hour
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.uintTob32(4001), 1, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getTimestampIndexByTimestamp(ETH_QUERY_ID, blocky.timestamp)).to.equal(1)
		expect(await tellor.getTimestampbyQueryIdandIndex(ETH_QUERY_ID, 1)).to.equal(blocky.timestamp)
		expect(await tellor.retrieveData(ETH_QUERY_ID, blocky.timestamp)).to.equal(h.uintTob32(4001))
		expect(await tellor.getReporterByTimestamp(ETH_QUERY_ID, blocky.timestamp)).to.equal(accounts[1].address)
		expect(await tellor.timeOfLastNewValue()).to.equal(blocky.timestamp)
		expect(await tellor.getReportsSubmittedByAddress(accounts[1].address)).to.equal(2)

		// Test submit multiple identical values w/ min _nonce
		await token.mint(accounts[2].address, h.toWei("120"))
		await token.connect(accounts[2]).approve(tellor.address, h.toWei("120"))
		await tellor.connect(accounts[2]).depositStake(web3.utils.toWei("120"))
		await tellor.connect(accounts[2]).submitValue(ETH_QUERY_ID, h.uintTob32(4001), 0, ETH_QUERY_DATA)
		await h.advanceTime(3600)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.uintTob32(4001), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getTimestampIndexByTimestamp(ETH_QUERY_ID, blocky.timestamp)).to.equal(3)
		expect(await tellor.getTimestampbyQueryIdandIndex(ETH_QUERY_ID, 3)).to.equal(blocky.timestamp)
		expect(await tellor.retrieveData(ETH_QUERY_ID, blocky.timestamp)).to.equal(h.uintTob32(4001))
		expect(await tellor.getReporterByTimestamp(ETH_QUERY_ID, blocky.timestamp)).to.equal(accounts[1].address)
		expect(await tellor.timeOfLastNewValue()).to.equal(blocky.timestamp)
		expect(await tellor.getReportsSubmittedByAddress(accounts[1].address)).to.equal(3)

		// Test max val for _nonce
		await h.advanceTime(3600)
		await expect(tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.uintTob32(4001), ethers.constants.MaxUint256, ETH_QUERY_DATA)).to.be.revertedWith("nonce must match timestamp index")

	})

	it("withdrawStake", async function () {
		await token.connect(accounts[1]).transfer(tellor.address, web3.utils.toWei("100"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("100"))
		expect(await tellor.getTotalStakers()).to.equal(1)

		await h.expectThrow(tellor.connect(accounts[1]).withdrawStake()) // test require: reporter not locked for withdrawal
		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("10"))
		await h.expectThrow(tellor.connect(accounts[1]).withdrawStake()) // test require: 7 days didn't pass
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(h.toWei("90"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(h.toWei("10"))

		await h.advanceTime(60 * 60 * 24 * 7)
		expect(await token.balanceOf(accounts[1].address)).to.equal(h.toWei("800"))
		await tellor.connect(accounts[1]).withdrawStake()
		expect(await token.balanceOf(accounts[1].address)).to.equal(h.toWei("810"))
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.stakedBalance]).to.equal(h.toWei("90"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(0)
		await h.expectThrow(tellor.connect(accounts[1]).withdrawStake()) // test require: reporter not locked for withdrawal
	})

	it("getCurrentValue", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		expect(await tellor.getCurrentValue(ETH_QUERY_ID)).to.equal(h.uintTob32(4000))
	})

	it("getGovernanceAddress", async function () {
		expect(await tellor.getGovernanceAddress()).to.equal(governance.address)
	})

	it("getNewValueCountbyQueryId", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		expect(await tellor.getNewValueCountbyQueryId(ETH_QUERY_ID)).to.equal(2)
	})

	it("getReportDetails", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky1 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4001), 0, ETH_QUERY_DATA)
		blocky2 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4002), 0, ETH_QUERY_DATA)
		blocky3 = await h.getBlock()
		await tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky3.timestamp)
		reportDetails = await tellor.getReportDetails(ETH_QUERY_ID, blocky1.timestamp)
		expect(reportDetails[0]).to.equal(accounts[1].address)
		expect(reportDetails[1]).to.equal(false)
		reportDetails = await tellor.getReportDetails(ETH_QUERY_ID, blocky2.timestamp)
		expect(reportDetails[0]).to.equal(accounts[1].address)
		expect(reportDetails[1]).to.equal(false)
		reportDetails = await tellor.getReportDetails(ETH_QUERY_ID, blocky3.timestamp)
		expect(reportDetails[0]).to.equal(accounts[1].address)
		expect(reportDetails[1]).to.equal(true)
		reportDetails = await tellor.getReportDetails(h.uintTob32(2), blocky1.timestamp)
		expect(reportDetails[0]).to.equal(h.zeroAddress)
		expect(reportDetails[1]).to.equal(false)
	})

	it("getReportingLock", async function () {
		expect(await tellor.getReportingLock()).to.equal(REPORTING_LOCK)
	})

	it("getReporterByTimestamp", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		expect(await tellor.getNewValueCountbyQueryId(ETH_QUERY_ID)).to.equal(1)
	})

	it("getReporterLastTimestamp", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getReporterLastTimestamp(accounts[1].address)).to.equal(blocky.timestamp)
	})

	it("getReportsSubmittedByAddress", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getReportsSubmittedByAddress(accounts[1].address)).to.equal(2)
	})

	it("getStakeAmount", async function () {
		expect(await tellor.getStakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)
	})

	it("getStakerInfo", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(h.toWei("1000"))
		await tellor.requestStakingWithdraw(h.toWei("100"))
		blocky = await h.getBlock()
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky2 = await h.getBlock()
		stakerDetails = await tellor.getStakerInfo(accounts[1].address)
		expect(stakerDetails[smap.startDate]).to.equal(blocky.timestamp)
		expect(stakerDetails[smap.stakedBalance]).to.equal(h.toWei("900"))
		expect(stakerDetails[smap.lockedBalance]).to.equal(h.toWei("100"))
		expect(stakerDetails[smap.rewardDebt]).to.equal(0)
		expect(stakerDetails[smap.reporterLastTimestamp]).to.equal(blocky2.timestamp)
		expect(stakerDetails[smap.startVoteCount]).to.equal(0)
		expect(stakerDetails[smap.reportsSubmitted]).to.equal(1)
		expect(stakerDetails[smap.startVoteTally]).to.equal(0)
	})

	it("getTimeOfLastNewValue", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getTimeOfLastNewValue()).to.equal(blocky.timestamp)
	})

	it("getTimestampbyQueryIdandIndex", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getTimestampbyQueryIdandIndex(ETH_QUERY_ID, 1)).to.equal(blocky.timestamp)
	})

	it("getTimestampIndexByTimestamp", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.getTimestampIndexByTimestamp(ETH_QUERY_ID, blocky.timestamp)).to.equal(1)
	})

	it("getTotalStakeAmount", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(h.toWei("100"))
		await tellor.requestStakingWithdraw(h.toWei("10"))
		expect(await tellor.getTotalStakeAmount()).to.equal(h.toWei("90"))
	})

	it("getTokenAddress", async function () {
		expect(await tellor.getTokenAddress()).to.equal(token.address)
	})

	it("getTotalStakers", async function () {
		tellor = await tellor.connect(accounts[1])

		// Only count unique stakers
		expect(await tellor.getTotalStakers()).to.equal(0)
		await tellor.depositStake(h.toWei("100"))
		expect(await tellor.getTotalStakers()).to.equal(1)
		await tellor.depositStake(h.toWei("100"))
		expect(await tellor.getTotalStakers()).to.equal(1)

		// Unstake, restake
		await tellor.connect(accounts[1]).requestStakingWithdraw(web3.utils.toWei("200"))
		expect(await tellor.totalStakers()).to.equal(0)
		await tellor.depositStake(h.toWei("100"))
		expect(await tellor.totalStakers()).to.equal(1)
	})

	it("retrieveData", async function () {
		tellor = await tellor.connect(accounts[1])
		await tellor.depositStake(web3.utils.toWei("100"))
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4000), 0, ETH_QUERY_DATA)
		await h.advanceTime(60 * 60 * 12)
		await tellor.submitValue(ETH_QUERY_ID, h.uintTob32(4001), 0, ETH_QUERY_DATA)
		blocky = await h.getBlock()
		expect(await tellor.retrieveData(ETH_QUERY_ID, blocky.timestamp)).to.equal(h.uintTob32(4001))

		// Test max/min values for _timestamp arg
		expect(await tellor.retrieveData(ETH_QUERY_ID, 0)).to.equal(ethers.utils.hexlify("0x"))
		expect(await tellor.retrieveData(ETH_QUERY_ID, ethers.constants.MaxUint256)).to.equal(ethers.utils.hexlify("0x"))
	})

	it("getTotalTimeBasedRewardsBalance", async function () {
		expect(BN(await tellor.getTotalTimeBasedRewardsBalance())).to.equal(0)
		await token.connect(accounts[1]).transfer(tellor.address, web3.utils.toWei("100"))
		expect(BN(await tellor.getTotalTimeBasedRewardsBalance())).to.equal(web3.utils.toWei("100"))
	})

	it("addStakingRewards", async function () {
		await token.mint(accounts[2].address, h.toWei("1000"))
		await h.expectThrow(tellor.connect(accounts[2]).addStakingRewards(h.toWei("1000"))) // test require: token.transferFrom...

		await token.connect(accounts[2]).approve(tellor.address, h.toWei("1000"))
		expect(await token.balanceOf(accounts[2].address)).to.equal(h.toWei("1000"))
		await tellor.connect(accounts[2]).addStakingRewards(h.toWei("1000"))
		expect(await tellor.stakingRewardsBalance()).to.equal(h.toWei("1000"))
		expect(await token.balanceOf(accounts[2].address)).to.equal(0)
		expect(await token.balanceOf(tellor.address)).to.equal(h.toWei("1000"))
		expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
		expect(await tellor.rewardRate()).to.equal(expectedRewardRate)

		// Test min value
		await tellor.connect(accounts[2]).addStakingRewards(0)
		expect(await tellor.stakingRewardsBalance()).to.equal(h.toWei("1000"))
		expect(await token.balanceOf(accounts[2].address)).to.equal(0)
		expect(await token.balanceOf(tellor.address)).to.equal(h.toWei("1000"))
		expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
		expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
	})

	it("getPendingRewardByStaker", async function () {
		let val = await tellor.callStatic.getPendingRewardByStaker(accounts[1].address)
		expect(val).to.equal(0)
		await token.mint(accounts[0].address, web3.utils.toWei("1000"))
		await token.approve(tellor.address, web3.utils.toWei("1000"))
		// add staking rewards
		await tellor.addStakingRewards(web3.utils.toWei("1000"))
		expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
		blocky0 = await h.getBlock()
		// advance time
		await h.advanceTime(86400 * 10)
		pendingReward = await tellor.callStatic.getPendingRewardByStaker(accounts[1].address)
		blocky1 = await h.getBlock()
		expectedAccumulatedRewardPerShare = BN(blocky1.timestamp - blocky0.timestamp).mul(expectedRewardRate).div(10)
		expectedPendingReward = BN(h.toWei("10")).mul(expectedAccumulatedRewardPerShare).div(h.toWei("1"))
		expect(pendingReward).to.equal(expectedPendingReward)
		// create 2 disputes, vote on 1
		await governance.beginDisputeMock()
		await governance.beginDisputeMock()
		await governance.connect(accounts[1]).voteMock(1)
		pendingReward = await tellor.callStatic.getPendingRewardByStaker(accounts[1].address)
		blocky2 = await h.getBlock()
		expectedAccumulatedRewardPerShare = BN(blocky2.timestamp - blocky0.timestamp).mul(expectedRewardRate).div(10)
		expectedPendingReward = BN(h.toWei("10")).mul(expectedAccumulatedRewardPerShare).div(h.toWei("1")).div(2)
		expect(pendingReward).to.equal(expectedPendingReward)
		expect(await tellor.callStatic.getPendingRewardByStaker(accounts[2].address)).to.equal(0)
	})

	it("getIndexForDataBefore()", async function () {
		// Setup
		await token.mint(accounts[1].address, web3.utils.toWei("1000"));
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("1000"))

		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(100), 0, BTC_QUERY_DATA)
		blocky0 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(100), 1, BTC_QUERY_DATA)
		blocky1 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(100), 2, BTC_QUERY_DATA)
		blocky2 = await h.getBlock()
		
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		// advance time one year and test
		await h.advanceTime(86400 * 365)
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		// advance time one year and test
		await h.advanceTime(86400 * 365)
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		for(i = 0; i < 50; i++) {
			await h.advanceTime(60 * 60 * 12)
			await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(100 + i), 0, BTC_QUERY_DATA)
		}
		blocky52 = await h.getBlock()
		
		// test last value disputed
		await tellor.connect(govSigner).removeValue(BTC_QUERY_ID, blocky52.timestamp)
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky52.timestamp + 1)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(51)

		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp + 1)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(2)

		// remove value at index 2
		await tellor.connect(govSigner).removeValue(BTC_QUERY_ID, blocky2.timestamp)
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp + 1)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky1.timestamp + 1)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(1)

		await tellor.connect(govSigner).removeValue(BTC_QUERY_ID, blocky1.timestamp)
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp - 1)
		expect(index[0]).to.be.true
		expect(index[1]).to.equal(0)

		await tellor.connect(govSigner).removeValue(BTC_QUERY_ID, blocky0.timestamp)
		index = await tellor.getIndexForDataBefore(BTC_QUERY_ID, blocky2.timestamp - 1)
		expect(index[0]).to.be.false
		expect(index[1]).to.equal(0)

		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
		blocky0 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
		blocky1 = await h.getBlock()

		await tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky0.timestamp)
		await tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky1.timestamp)

		index = await tellor.getIndexForDataBefore(ETH_QUERY_ID, blocky1.timestamp + 1)
		expect(index[0]).to.be.false
		expect(index[1]).to.equal(0)

		index = await tellor.getIndexForDataBefore(ETH_QUERY_ID, blocky0.timestamp + 1)
		expect(index[0]).to.be.false
		expect(index[1]).to.equal(0)

		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
		blocky2 = await h.getBlock()

		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
		// blocky3 = await h.getBlock()

		await tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky2.timestamp)
		index = await tellor.getIndexForDataBefore(ETH_QUERY_ID, blocky2.timestamp + 1)
		expect(index[0]).to.be.false
		expect(index[1]).to.equal(0)
	})

	it("getDataBefore()", async function () {
		// Setup
		await token.mint(accounts[1].address, web3.utils.toWei("1000"));
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("1000"))

		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(150), 0, BTC_QUERY_DATA)
		blocky1 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(160), 1, BTC_QUERY_DATA)
		blocky2 = await h.getBlock()
		await h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(170), 2, BTC_QUERY_DATA)
		blocky3 = await h.getBlock()

		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky3.timestamp + 1)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(170))
		expect(dataBefore[2]).to.equal(blocky3.timestamp)

		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(150))
		expect(dataBefore[2]).to.equal(blocky1.timestamp)

		// advance time one year and test
		await h.advanceTime(86400 * 365)
		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky3.timestamp + 1)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(170))
		expect(dataBefore[2]).to.equal(blocky3.timestamp)

		// advance time one year and test
		await h.advanceTime(86400 * 365)
		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky3.timestamp + 1)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(170))
		expect(dataBefore[2]).to.equal(blocky3.timestamp)

		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(150))
		expect(dataBefore[2]).to.equal(blocky1.timestamp)

		// submit 50 values and test
		for(i = 0; i < 50; i++) {
			await tellor.connect(accounts[1]).submitValue(BTC_QUERY_ID, h.bytes(100 + i), 0, BTC_QUERY_DATA)
			await h.advanceTime(60 * 60 * 12)
		}

		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky3.timestamp + 1)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(170))
		expect(dataBefore[2]).to.equal(blocky3.timestamp)

		dataBefore = await tellor.getDataBefore(BTC_QUERY_ID, blocky2.timestamp)
		expect(dataBefore[0])
		expect(dataBefore[1]).to.equal(h.bytes(150))
		expect(dataBefore[2]).to.equal(blocky1.timestamp)
	})
	
	it("verify()", async function() {
		expect(await tellor.verify()).to.equal(9999)
	})

	it("updateStakeAmount()", async function () {
		// Setup
		await token.mint(accounts[1].address, web3.utils.toWei("10000"));
		await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("10000"))
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("10000"))
		
		// Test no reported TRB price
		await tellor.updateStakeAmount()
		expect(await tellor.stakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)

		// Test updating when 12 hrs have NOT passed
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 2), 0, TRB_QUERY_DATA)
		await tellor.connect(accounts[1]).updateStakeAmount()
		expect(await tellor.getStakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)

		// Test updating when 12 hrs have passed
		h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).updateStakeAmount()
		expect(await tellor.getStakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)

		// Test updating when multiple prices have been reported
		h.advanceTime(60 * 60 * 1)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 1.5), 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 1)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 2), 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 1)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 3), 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).updateStakeAmount()
		expect(await tellor.getStakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)

		// Test bad TRB price encoding
		badPrice = abiCoder.encode(["string"], ["Where's the beef?"])
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, badPrice, 0, TRB_QUERY_DATA)
		await h.advanceTime(86400/2)
		await h.expectThrow(tellor.updateStakeAmount())
		expect(await tellor.stakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)

		// Test reported TRB price outside limits - high
		highPrice = h.toWei("1000001")
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(highPrice), 0, TRB_QUERY_DATA)
		await h.advanceTime(86400/2)
		await h.expectThrow(tellor.updateStakeAmount())
		expect(await tellor.stakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT) 

		// Test reported TRB price outside limits - low
		lowPrice = h.toWei("0.009")
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(lowPrice), 0, TRB_QUERY_DATA)
		await h.advanceTime(86400/2)
		await h.expectThrow(tellor.updateStakeAmount())
		expect(await tellor.stakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT) 

		// Test updating when multiple prices have been reported
		h.advanceTime(60 * 60 * 1)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 7), 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 1)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 8), 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 1)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(PRICE_TRB * 9), 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).updateStakeAmount()
		expect(await tellor.getStakeAmount()).to.equal(MINIMUM_STAKE_AMOUNT)

		// Test updating when price is below inflection price
		inflectionPrice = h.toWei("5") // BigInt(STAKE_AMOUNT_USD_TARGET) * BigInt(1e18) / BigInt(MINIMUM_STAKE_AMOUNT)
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, h.uintTob32(inflectionPrice / 2), 0, TRB_QUERY_DATA) // price = inflectionPrice / 2
		await h.advanceTime(60 * 60 * 12)
		await tellor.updateStakeAmount()
		expStakeAmount = BigInt(MINIMUM_STAKE_AMOUNT) * BigInt(2)
		expect(await tellor.getStakeAmount()).to.equal(BigInt(MINIMUM_STAKE_AMOUNT) * BigInt(2))

		// Test updating when multiple prices have been reported
		h.advanceTime(60 * 60 * 1)
		reportedPrice = BigInt(inflectionPrice) / BigInt(1)
		encodedPrice = abiCoder.encode(['uint256'], [reportedPrice])
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, encodedPrice, 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 1)
		reportedPrice = BigInt(inflectionPrice) / BigInt(2)
		encodedPrice = abiCoder.encode(['uint256'], [reportedPrice])
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, encodedPrice, 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 1)
		reportedPrice = BigInt(inflectionPrice) / BigInt(3)
		encodedPrice = abiCoder.encode(['uint256'], [reportedPrice])
		await tellor.connect(accounts[1]).submitValue(TRB_QUERY_ID, encodedPrice, 0, TRB_QUERY_DATA)
		h.advanceTime(60 * 60 * 12)
		await tellor.connect(accounts[1]).updateStakeAmount()
		expStakeAmount = BigInt(STAKE_AMOUNT_USD_TARGET) * BigInt(1e18) / (BigInt(inflectionPrice) / BigInt(3))
		expect(await tellor.getStakeAmount()).to.equal(expStakeAmount)
	})

	it("_updateRewards()", async function () {
		// set up
		expTotalStakeAmount = BigInt(0)
		depositStakeAmount = h.toWei("50")

		// update rewards
		await tellor.updateRewards()
		blocky0 = await h.getBlock()

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky0.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		expect(await tellor.rewardRate()).to.equal(0)

		// deposit a stake

		await tellor.connect(accounts[1]).depositStake(depositStakeAmount)
		blocky0 = await h.getBlock()
		expTotalStakeAmount += BigInt(depositStakeAmount)

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky0.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		expect(await tellor.rewardRate()).to.equal(0)

		// deposit another stake
		await tellor.connect(accounts[1]).depositStake(depositStakeAmount)
		blocky0 = await h.getBlock()
		expTotalStakeAmount += BigInt(depositStakeAmount)

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky0.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		expect(await tellor.rewardRate()).to.equal(0)

		// add staking rewards
		expect(await tellor.stakingRewardsBalance()).to.equal(0)
		const STAKING_REWARDS1 = h.toWei("1000")
		await token.mint(accounts[0].address, STAKING_REWARDS1)
		await token.approve(tellor.address, STAKING_REWARDS1)
		await tellor.addStakingRewards(STAKING_REWARDS1)
		blocky1 = await h.getBlock()

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky1.timestamp)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
		expect(await tellor.stakingRewardsBalance()).to.equal(STAKING_REWARDS1)
		expect(await tellor.totalRewardDebt()).to.equal(0)
		expectedRewardRate = Math.floor(STAKING_REWARDS1 / (86400 * 30))
		expect(await tellor.rewardRate()).to.equal(expectedRewardRate)

		// advance time 1 day
		await h.advanceTime(86400)

		// updateRewards
		await tellor.updateRewards()
		blocky2 = await h.getBlock()

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky2.timestamp)
		expect(await tellor.stakingRewardsBalance()).to.equal(STAKING_REWARDS1)
		expect(await tellor.totalRewardDebt()).to.equal(0)
		expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
		expAccumRewPerShare = BigInt(blocky2.timestamp - blocky1.timestamp) * BigInt(expectedRewardRate) * BigInt(1e18) / BigInt(100e18)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(expAccumRewPerShare)

		// deposit another stake
		expect(await token.balanceOf(accounts[1].address)).to.equal(h.toWei("900"))
		await tellor.connect(accounts[1]).depositStake(depositStakeAmount)
		blocky3 = await h.getBlock()
		expTotalStakeAmount += BigInt(depositStakeAmount)

		// state checks
		expAccumRewPerShare = expAccumRewPerShare + (BigInt(blocky3.timestamp - blocky2.timestamp)) * BigInt(expectedRewardRate) * BigInt(1e18) / BigInt(100e18)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(expAccumRewPerShare)
		expectedPayout = expAccumRewPerShare * BigInt(100e18) / BigInt(1e18)
		expectedBal = BigInt(h.toWei("900")) - BigInt(h.toWei("50")) + expectedPayout
		expect(await token.balanceOf(accounts[1].address)).to.equal(expectedBal)
		expTotalRewardDebt = BigInt(150e18) * expAccumRewPerShare / BigInt(1e18)
		expect(await tellor.totalRewardDebt()).to.equal(expTotalRewardDebt)
		expStakingRewardsBal = BigInt(STAKING_REWARDS1) - expectedPayout
		expect(await tellor.stakingRewardsBalance()).to.equal(expStakingRewardsBal)

		// update rewards
		await tellor.updateRewards()
		blocky4 = await h.getBlock()

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky4.timestamp)
		expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
		expAccumRewPerShare = expAccumRewPerShare + (BigInt(blocky4.timestamp - blocky3.timestamp) * BigInt(expectedRewardRate) * BigInt(1e18) / BigInt(150e18))
		expect(await tellor.accumulatedRewardPerShare()).to.equal(expAccumRewPerShare)
		expect(await tellor.stakingRewardsBalance()).to.equal(expStakingRewardsBal) // shouldn't change

		// advance time 30 days
		await h.advanceTime(86400 * 30)

		// calculate real new pending rewards, which should be the same value used in the following 
		// updateRewards() call 
		realStakingRewardsBalance = BigInt(await tellor.stakingRewardsBalance())
		realAccumulatedRewardPerShare = BigInt(await tellor.accumulatedRewardPerShare())
		realTotalStakeAmount = BigInt(await tellor.totalStakeAmount())
		realTotalRewardDebt = BigInt(await tellor.totalRewardDebt())
		realNewPendingRewards = realStakingRewardsBalance - ((realAccumulatedRewardPerShare * realTotalStakeAmount) / BigInt(1e18) - realTotalRewardDebt)

		// update rewards
		await tellor.updateRewards()
		blocky5 = await h.getBlock()

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky5.timestamp)
		expect(await tellor.rewardRate()).to.equal(0) // rewards ran out, reward rate should be 0
		expNewPendingRewards = expStakingRewardsBal - ((expAccumRewPerShare * expTotalStakeAmount) / BigInt(1e18) - expTotalRewardDebt)
		expAccumRewPerShare = expAccumRewPerShare + (expNewPendingRewards * BigInt(1e18)) / BigInt(150e18)
		expect(await tellor.accumulatedRewardPerShare()).to.equal(expAccumRewPerShare)
		expect(realNewPendingRewards).to.equal(expNewPendingRewards)

		// advance time 1 day
		await h.advanceTime(86400)

		// update rewards
		await tellor.updateRewards()
		blocky5 = await h.getBlock()

		// state checks
		expect(await tellor.timeOfLastAllocation()).to.equal(blocky5.timestamp) // should update to latest updateRewards ts
		expect(await tellor.rewardRate()).to.equal(0) // should still be zero
		expect(await tellor.accumulatedRewardPerShare()).to.equal(expAccumRewPerShare) // shouldn't change
	})

	it("_updateStakeAndPayRewards", async function () {
        await token.mint(accounts[0].address, web3.utils.toWei("1000"))
        await token.approve(tellor.address, web3.utils.toWei("1000"))
        // check initial conditions
        expect(await tellor.stakingRewardsBalance()).to.equal(0)
        expect(await tellor.rewardRate()).to.equal(0)
        // add staking rewards
        await tellor.addStakingRewards(web3.utils.toWei("1000"))
        // check conditions after adding rewards
        expect(await tellor.stakingRewardsBalance()).to.equal(web3.utils.toWei("1000"))
        expect(await tellor.totalRewardDebt()).to.equal(0)
        expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
        expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
        // create 2 mock disputes, vote once
        await governance.beginDisputeMock()
        await governance.beginDisputeMock()
        await governance.connect(accounts[1]).voteMock(1)
        // deposit stake
        await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
        blocky0 = await h.getBlock()
        // check conditions after depositing stake
        expect(await tellor.stakingRewardsBalance()).to.equal(web3.utils.toWei("1000"))
        expect(await tellor.getTotalStakeAmount()).to.equal(web3.utils.toWei("10"))
        expect(await tellor.totalRewardDebt()).to.equal(0)
        expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
        expect(await tellor.timeOfLastAllocation()).to.equal(blocky0.timestamp)
        stakerInfo = await tellor.getStakerInfo(accounts[1].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(web3.utils.toWei("10")) // staked balance
        expect(stakerInfo[smap.rewardDebt]).to.equal(0) // rewardDebt
        expect(stakerInfo[smap.startVoteCount]).to.equal(2) // startVoteCount
        expect(stakerInfo[smap.startVoteTally]).to.equal(1) // startVoteTally
        // advance time
        await h.advanceTime(86400 * 10)
        expect(await token.balanceOf(accounts[1].address)).to.equal(h.toWei("990"))
        // deposit 0 stake, update rewards
        await tellor.connect(accounts[1]).depositStake(0)
        blocky1 = await h.getBlock()
        // check conditions after updating rewards
        expect(await tellor.timeOfLastAllocation()).to.equal(blocky1.timestamp)
        expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
        expectedAccumulatedRewardPerShare = BN(blocky1.timestamp - blocky0.timestamp).mul(expectedRewardRate).div(10)
        expectedBalance = BN(h.toWei("10")).mul(expectedAccumulatedRewardPerShare).div(h.toWei("1")).add(h.toWei("990"))
        expect(await token.balanceOf(accounts[1].address)).to.equal(expectedBalance)
        expect(await tellor.accumulatedRewardPerShare()).to.equal(expectedAccumulatedRewardPerShare)
        expect(await tellor.totalRewardDebt()).to.equal(expectedBalance.sub(h.toWei("990")))
        stakerInfo = await tellor.getStakerInfo(accounts[1].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(h.toWei("10")) // staked balance
        expect(stakerInfo[smap.rewardDebt]).to.equal(expectedBalance.sub(h.toWei("990"))) // rewardDebt
        expect(stakerInfo[smap.startVoteCount]).to.equal(2) // startVoteCount
        expect(stakerInfo[smap.startVoteTally]).to.equal(1) // startVoteTally
        // start a dispute
        await governance.beginDisputeMock()
        // advance time
        await h.advanceTime(86400 * 10)
        // deposit 0 stake, update rewards
        await tellor.connect(accounts[1]).depositStake(0)
        blocky2 = await h.getBlock()
        // check conditions after updating rewards
        expect(await tellor.timeOfLastAllocation()).to.equal(blocky2.timestamp)
        expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
        expectedAccumulatedRewardPerShare = BN(blocky2.timestamp - blocky1.timestamp).mul(expectedRewardRate).div(10).add(expectedAccumulatedRewardPerShare)
        expect(await token.balanceOf(accounts[1].address)).to.equal(expectedBalance)
        expect(await tellor.accumulatedRewardPerShare()).to.equal(expectedAccumulatedRewardPerShare)
        expectedRewardDebt = expectedAccumulatedRewardPerShare.mul(10)
        expect(await tellor.totalRewardDebt()).to.equal(expectedRewardDebt)
        stakerInfo = await tellor.getStakerInfo(accounts[1].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(h.toWei("10")) // staked balance
        expect(stakerInfo[smap.rewardDebt]).to.equal(expectedRewardDebt) // rewardDebt
        expect(stakerInfo[smap.startVoteCount]).to.equal(2) // startVoteCount
        expect(stakerInfo[smap.startVoteTally]).to.equal(1) // startVoteTally
        // start a dispute and vote
        await governance.beginDisputeMock()
        await governance.connect(accounts[1]).voteMock(4)
        // advance time
        await h.advanceTime(86400 * 5)
        // deposit 0 stake, update rewards
        await tellor.connect(accounts[1]).depositStake(0)
        blocky3 = await h.getBlock()
        // check conditions after updating rewards
        expect(await tellor.timeOfLastAllocation()).to.equal(blocky3.timestamp)
        expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
        expectedAccumulatedRewardPerShare = BN(blocky3.timestamp - blocky2.timestamp).mul(expectedRewardRate).div(10).add(expectedAccumulatedRewardPerShare)
        expectedBalance = expectedBalance.add(expectedAccumulatedRewardPerShare.mul(10).sub(expectedRewardDebt).div(2))
        expect(await token.balanceOf(accounts[1].address)).to.equal(expectedBalance)
        expect(await tellor.accumulatedRewardPerShare()).to.equal(expectedAccumulatedRewardPerShare)
        expectedRewardDebt = expectedAccumulatedRewardPerShare.mul(10)
        expect(await tellor.totalRewardDebt()).to.equal(expectedRewardDebt)
        stakerInfo = await tellor.getStakerInfo(accounts[1].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(h.toWei("10")) // staked balance
        expect(stakerInfo[smap.rewardDebt]).to.equal(expectedRewardDebt) // rewardDebt
        expect(stakerInfo[smap.startVoteCount]).to.equal(2) // startVoteCount
        expect(stakerInfo[smap.startVoteTally]).to.equal(1) // startVoteTally
        expect(await tellor.stakingRewardsBalance()).to.equal(BN(h.toWei("1000")).sub(expectedBalance).add(h.toWei("990")))
    })

	it("getRealStakingRewardsBalance", async function () {
		expect(await tellor.getRealStakingRewardsBalance()).to.equal(0)
		await token.mint(accounts[0].address, web3.utils.toWei("1000"))
		await token.approve(tellor.address, web3.utils.toWei("1000"))
		// add staking rewards
		await tellor.addStakingRewards(web3.utils.toWei("1000"))
		expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
		await tellor.connect(accounts[1]).depositStake(web3.utils.toWei("10"))
		blocky0 = await h.getBlock()
		// advance time
		await h.advanceTime(86400 * 10)
		pendingReward = await tellor.getPendingRewardByStaker(accounts[1].address)
		blocky1 = await h.getBlock()
		expectedAccumulatedRewardPerShare = BN(blocky1.timestamp - blocky0.timestamp).mul(expectedRewardRate).div(10)
		expectedPendingReward = BN(h.toWei("10")).mul(expectedAccumulatedRewardPerShare).div(h.toWei("1"))
		expectedRealStakingRewardsBalance = BigInt(h.toWei("1000")) - BigInt(expectedPendingReward)
		expect(await tellor.getRealStakingRewardsBalance()).to.equal(expectedRealStakingRewardsBalance)
		await h.advanceTime(86400 * 30)
		expect(await tellor.getRealStakingRewardsBalance()).to.equal(0)
		expect(await tellor.stakingRewardsBalance()).to.equal(h.toWei("1000"))
	})
});