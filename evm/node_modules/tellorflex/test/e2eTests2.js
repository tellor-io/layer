const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const { assembleTargets } = require("solidity-coverage/plugins/resources/plugin.utils");
const BN = ethers.BigNumber.from

describe("TellorFlex - e2e Tests Two", function() {

	let tellor;
    let governance;
    let govSigner;
	let token;
	let accounts;
    let owner;
    const MINIMUM_STAKE_AMOUNT = web3.utils.toWei("100")
	const STAKE_AMOUNT_USD_TARGET = h.toWei("500");
    const PRICE_TRB = h.toWei("50");
	const REQUIRED_STAKE = h.toWei((parseInt(web3.utils.fromWei(STAKE_AMOUNT_USD_TARGET)) / parseInt(web3.utils.fromWei(PRICE_TRB))).toString());
	const REPORTING_LOCK = 43200; // 12 hours
    const REWARD_RATE_TARGET = 60 * 60 * 24 * 30; // 30 days
    const abiCoder = new ethers.utils.AbiCoder
	const TRB_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["trb", "usd"])
	const TRB_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", TRB_QUERY_DATA_ARGS])
	const TRB_QUERY_ID = ethers.utils.keccak256(TRB_QUERY_DATA)
    const ETH_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["eth", "usd"])
    const ETH_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", ETH_QUERY_DATA_ARGS])
    const ETH_QUERY_ID = ethers.utils.keccak256(ETH_QUERY_DATA)

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
		const TellorFlex = await ethers.getContractFactory("TellorFlex");
		tellor = await TellorFlex.deploy(token.address, REPORTING_LOCK, STAKE_AMOUNT_USD_TARGET, PRICE_TRB, MINIMUM_STAKE_AMOUNT, TRB_QUERY_ID);
        owner = await ethers.getSigner(await tellor.owner())
		await tellor.deployed();
        await governance.setTellorAddress(tellor.address);
		await token.mint(accounts[1].address, h.toWei("1000"));
        await token.connect(accounts[1]).approve(tellor.address, h.toWei("1000"))
        await hre.network.provider.request({
            method: "hardhat_impersonateAccount",
            params: [governance.address]}
        )
        govSigner = await ethers.getSigner(governance.address);
        await accounts[10].sendTransaction({to:governance.address,value:ethers.utils.parseEther("1.0")}); 

        await tellor.connect(owner).init(governance.address)
	});
    it("Staked multiple times, disputed but keeps reporting", async function() {
        await tellor.connect(accounts[1]).depositStake(h.toWei("300"))
        await tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA)
		let blocky = await h.getBlock()
		expect(await tellor.getNewValueCountbyQueryId(ETH_QUERY_ID)).to.equal(1)
		expect(await tellor.retrieveData(ETH_QUERY_ID, blocky.timestamp)).to.equal(h.bytes(100))
		await h.expectThrow(tellor.connect(accounts[1]).removeValue(ETH_QUERY_ID, blocky.timestamp)) // only gov can removeValue
		await tellor.connect(govSigner).removeValue(ETH_QUERY_ID, blocky.timestamp)
        await tellor.connect(govSigner).slashReporter(accounts[1].address, accounts[2].address)
        await h.advanceTime(86400/2/3)
        await h.expectThrow(tellor.connect(accounts[1]).submitValue(ETH_QUERY_ID, h.bytes(100), 0, ETH_QUERY_DATA))
        await h.advanceTime(86400/2/3)
        let vars = await tellor.getStakerInfo(accounts[1].address)
        assert(vars[1] == h.toWei("200"), "should still have money staked")
    })
    it("Realistic test with staking rewards and disputes", async function() {
        await token.mint(accounts[0].address, h.toWei("1000"))
        await token.approve(tellor.address, h.toWei("1000"))
        // check initial conditions
        expect(await tellor.stakingRewardsBalance()).to.equal(0)
        expect(await tellor.rewardRate()).to.equal(0)
        // add staking rewards
        await tellor.addStakingRewards(h.toWei("1000"))
        // check conditions after adding rewards
        expect(await tellor.stakingRewardsBalance()).to.equal(h.toWei("1000"))
        expect(await tellor.totalRewardDebt()).to.equal(0)
        expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
        expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
        // create 2 mock disputes, vote once
        await governance.beginDisputeMock()
        await governance.beginDisputeMock()
        await governance.connect(accounts[1]).voteMock(1)
        // deposit stake
        await tellor.connect(accounts[1]).depositStake(h.toWei("10"))
        blocky0 = await h.getBlock()
        // check conditions after depositing stake
        expect(await tellor.stakingRewardsBalance()).to.equal(h.toWei("1000"))
        expect(await tellor.getTotalStakeAmount()).to.equal(h.toWei("10"))
        expect(await tellor.totalRewardDebt()).to.equal(0)
        expect(await tellor.accumulatedRewardPerShare()).to.equal(0)
        expect(await tellor.timeOfLastAllocation()).to.equal(blocky0.timestamp)
        stakerInfo = await tellor.getStakerInfo(accounts[1].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(h.toWei("10")) // staked balance
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
    it("two accounts stake (one 10 TRB one 20 TRB), does account2 have double reward debt?", async function() {
        await token.mint(accounts[0].address, h.toWei("1000"))
        await token.approve(tellor.address, h.toWei("1000"))
        // check initial conditions
        expect(await tellor.stakingRewardsBalance()).to.equal(0)
        expect(await tellor.rewardRate()).to.equal(0)
        expect(await tellor.totalRewardDebt()).to.equal(0)
        // add staking rewards
        await tellor.addStakingRewards(h.toWei("1000"))
        // check conditions after adding rewards
        expect(await tellor.stakingRewardsBalance()).to.equal(h.toWei("1000"))
        expect(await tellor.totalRewardDebt()).to.equal(0)
        expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
        expect(await tellor.rewardRate()).to.equal(expectedRewardRate)
        
        // deposit 2 stakes
        await tellor.connect(accounts[1]).depositStake(h.toWei("10"))
        blocky1 = await h.getBlock()
        await token.mint(accounts[2].address, h.toWei("20"))
        await token.connect(accounts[2]).approve(tellor.address, h.toWei("20"))
        await tellor.connect(accounts[2]).depositStake(h.toWei("20"))

        await h.advanceTime(86400 * 7)

        // update rewards by depositing 0 stake
        await tellor.connect(accounts[1]).depositStake(0)
        await tellor.connect(accounts[2]).depositStake(0)

        stakerInfo1 = await tellor.getStakerInfo(accounts[1].address)
        stakerInfo2 = await tellor.getStakerInfo(accounts[2].address)

        ratio = BigInt(stakerInfo2[smap.rewardDebt]) / BigInt(stakerInfo1[smap.rewardDebt])
        expect(ratio).to.equal(BigInt(2))
    })

    it("open dispute A when reporter stakes, reporter votes on A and one more B", async function() {
        // create 2 disputes
        await governance.beginDisputeMock()
        await governance.beginDisputeMock()
        // vote on one dispute
        await governance.connect(accounts[2]).voteMock(1)
        // deposit stake
        await token.mint(accounts[2].address, h.toWei("10"))
        await token.connect(accounts[2]).approve(tellor.address, h.toWei("100"))
        await tellor.connect(accounts[2]).depositStake(h.toWei("10"))
        // check staker info
        stakerInfo = await tellor.getStakerInfo(accounts[2].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(h.toWei("10")) // staked balance
        expect(stakerInfo[smap.startVoteCount]).to.equal(2) // startVoteCount
        expect(stakerInfo[smap.startVoteTally]).to.equal(1) // startVoteTally
        // start a dispute
        await governance.beginDisputeMock()
        // vote
        await governance.connect(accounts[2]).voteMock(2)
        await governance.connect(accounts[2]).voteMock(3)
        // deposit staking rewards
        await token.mint(accounts[0].address, h.toWei("1000"))
        await token.connect(accounts[0]).approve(tellor.address, h.toWei("1000"))
        await tellor.connect(accounts[0]).addStakingRewards(h.toWei("1000"))
        // advance time
        await h.advanceTime(86400 * 31)
        // withdraw stake
        await tellor.connect(accounts[2]).requestStakingWithdraw(h.toWei("10"))
        await h.advanceTime(86400 * 7)
        await tellor.connect(accounts[2]).withdrawStake()
        // check reporter bal
        expect(await token.balanceOf(accounts[2].address)).to.equal(h.toWei("1010"))

        // stake again, check staker info
        await tellor.connect(accounts[2]).depositStake(h.toWei("10"))
        stakerInfo = await tellor.getStakerInfo(accounts[2].address)
        expect(stakerInfo[smap.stakedBalance]).to.equal(h.toWei("10")) // staked balance
        expect(stakerInfo[smap.startVoteCount]).to.equal(3) // startVoteCount
        expect(stakerInfo[smap.startVoteTally]).to.equal(3) // startVoteTally
    })
    it("2 reporters stake and unstake with staking rewards", async function() {
        await token.mint(accounts[0].address, h.toWei("1000"))
        await token.mint(accounts[2].address, h.toWei("100"))
        await token.mint(accounts[3].address, h.toWei("100"))

        // deposit 1st stake
        await token.connect(accounts[2]).approve(tellor.address, h.toWei("100"))
        await tellor.connect(accounts[2]).depositStake(h.toWei("100"))
        blocky0 = await h.getBlock()

        expectedAccumulatedRewardPerShare = 0
        assert(await tellor.accumulatedRewardPerShare() == expectedAccumulatedRewardPerShare, "accumulatedRewardPerShare should be 0")

        stakerInfo = await tellor.getStakerInfo(accounts[2].address)
        expectedRewardDebt = 0
        assert(stakerInfo[smap.rewardDebt] == expectedRewardDebt, "reward debt should be correct") // rewardDebt
        assert(stakerInfo[smap.stakedBalance] == h.toWei("100"), "Staked balance should be correct") // staked balance

        expectedTotalRewardDebt = 0
        assert(await tellor.totalRewardDebt() == expectedTotalRewardDebt, "totalRewardDebt should be correct")

        // add staking rewards
        await token.connect(accounts[0]).approve(tellor.address, h.toWei("1000"))
        await tellor.connect(accounts[0]).addStakingRewards(h.toWei("1000"))
        blocky1 = await h.getBlock()

        expectedRewardRate = Math.floor(h.toWei("1000") / REWARD_RATE_TARGET)
        assert(await tellor.rewardRate() == expectedRewardRate, "rewardRate should be correct")

        expectedAccumulatedRewardPerShare = 0
        assert(await tellor.accumulatedRewardPerShare() == expectedAccumulatedRewardPerShare, "accumulatedRewardPerShare should be 0")

        expectedTotalRewardDebt = 0
        assert(await tellor.totalRewardDebt() == expectedTotalRewardDebt, "totalRewardDebt should be correct")

        // deposit 2nd stake
        await token.connect(accounts[3]).approve(tellor.address, h.toWei("100"))
        await tellor.connect(accounts[3]).depositStake(h.toWei("100"))
        blocky2 = await h.getBlock()

        expectedAccumulatedRewardPerShare = BigInt(blocky2.timestamp - blocky1.timestamp) * BigInt(expectedRewardRate) / BigInt(100)
        assert(await tellor.accumulatedRewardPerShare() == expectedAccumulatedRewardPerShare, "accumulatedRewardPerShare should be correct")

        stakerInfo = await tellor.getStakerInfo(accounts[3].address)
        expectedRewardDebt = BigInt(expectedAccumulatedRewardPerShare) * BigInt(100)
        assert(stakerInfo[smap.rewardDebt] == expectedRewardDebt, "rewardDebt should be correct")
        assert(stakerInfo[smap.stakedBalance] == h.toWei("100"), "Staked balance should be correct") 

        expectedTotalRewardDebt = expectedRewardDebt
        assert(await tellor.totalRewardDebt() == expectedTotalRewardDebt, "totalRewardDebt should be correct")

        // advance time
        await h.advanceTime(86400)

        // withdraw 1st stake
        await tellor.connect(accounts[2]).requestStakingWithdraw(h.toWei("100"))
        blocky3 = await h.getBlock()

        expectedAccumulatedRewardPerShare = BigInt(blocky3.timestamp - blocky2.timestamp) * BigInt(expectedRewardRate) / BigInt(200) + BigInt(expectedAccumulatedRewardPerShare)
        assert(await tellor.accumulatedRewardPerShare() == expectedAccumulatedRewardPerShare, "accumulatedRewardPerShare should be correct")

        expectedStakerBalance1 = BigInt(100) * BigInt(expectedAccumulatedRewardPerShare)
        assert(await token.balanceOf(accounts[2].address) == expectedStakerBalance1, "staker balance should be correct")
        stakerInfo = await tellor.getStakerInfo(accounts[2].address)
        assert(stakerInfo[smap.stakedBalance] == 0, "staked balance should be 0")
        assert(stakerInfo[smap.rewardDebt] == 0, "rewardDebt should be correct")

        assert(await tellor.totalRewardDebt() == expectedTotalRewardDebt, "totalRewardDebt should be correct")
        
        // withdraw 2nd stake
        await tellor.connect(accounts[3]).requestStakingWithdraw(h.toWei("100"))
        blocky4 = await h.getBlock()

        expectedAccumulatedRewardPerShare = BigInt(blocky4.timestamp - blocky3.timestamp) * BigInt(expectedRewardRate) / BigInt(100) + BigInt(expectedAccumulatedRewardPerShare)
        assert(await tellor.accumulatedRewardPerShare() == expectedAccumulatedRewardPerShare, "accumulatedRewardPerShare should be correct")

        expectedStakerBalance2 = BigInt(100) * BigInt(expectedAccumulatedRewardPerShare) - expectedRewardDebt
        assert(await token.balanceOf(accounts[3].address) == expectedStakerBalance2, "staker balance should be correct")
        stakerInfo = await tellor.getStakerInfo(accounts[3].address)
        assert(stakerInfo[smap.stakedBalance] == 0, "staked balance should be 0")
        assert(stakerInfo[smap.rewardDebt] == 0, "rewardDebt should be correct")

        assert(await tellor.totalRewardDebt() == 0, "totalRewardDebt should be correct")

        await h.advanceTime(86400 * 7)

        // fully withdraw both stakes
        await tellor.connect(accounts[2]).withdrawStake()
        await tellor.connect(accounts[3]).withdrawStake()

        expectedStakerBalance1 += BigInt(h.toWei("100"))
        assert(await token.balanceOf(accounts[2].address) == expectedStakerBalance1, "staker balance should be correct")
        expectedStakerBalance2 += BigInt(h.toWei("100"))
        assert(await token.balanceOf(accounts[3].address) == expectedStakerBalance2, "staker balance should be correct")
    })

    it("Time based rewards don't steal from stakes pending withdrawal", async function() {
		await token.mint(accounts[0].address, h.toWei("10000"))
        await token.approve(tellor.address, h.toWei("10000"))
        await tellor.depositStake(h.toWei("1000"))
        assert(await tellor.getTotalTimeBasedRewardsBalance() == 0, "total time based rewards balance should be 0")
        assert(await tellor.toWithdraw() == 0, "toWithdraw should be 0")
        await tellor.requestStakingWithdraw(h.toWei("100"))
        assert(await tellor.getTotalTimeBasedRewardsBalance() == 0, "total time based rewards balance should be 0")
        assert(await tellor.toWithdraw() == h.toWei("100"), "toWithdraw should be correct")

        await tellor.submitValue(h.hash(h.uintTob32(1)), h.uintTob32(1000), 0, h.uintTob32(1))
        await tellor.connect(govSigner).slashReporter(accounts[0].address, governance.address)

        assert(await tellor.getTotalTimeBasedRewardsBalance() == 0, "total time based rewards balance should be 0")
        assert(await tellor.toWithdraw() == 0, "toWithdraw should be correct")
	})
})