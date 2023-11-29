const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const BN = ethers.BigNumber.from

describe("TellorFlex - e2e Tests Three", function() {

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
	});

    it("Cannot deposit stake if governance address not set", async function() {
        await h.expectThrow(tellor.connect(accounts[1]).depositStake(h.toWei("10")));
        await tellor.connect(owner).init(governance.address)
        await tellor.connect(accounts[1]).depositStake(h.toWei("10"));
    })
   
})