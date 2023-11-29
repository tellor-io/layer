const { expect } = require("chai");
const { network, ethers } = require("hardhat");
const h = require("./helpers/helpers");
const web3 = require('web3');
const BN = ethers.BigNumber.from

describe("TellorFlex - Function Tests Init Function", function() {

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
    const abiCoder = new ethers.utils.AbiCoder
	const TRB_QUERY_DATA_ARGS = abiCoder.encode(["string", "string"], ["trb", "usd"])
	const TRB_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["SpotPrice", TRB_QUERY_DATA_ARGS])
	const TRB_QUERY_ID = ethers.utils.keccak256(TRB_QUERY_DATA)

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
		await token.mint(accounts[1].address, web3.utils.toWei("1000"));
        await token.connect(accounts[1]).approve(tellor.address, web3.utils.toWei("1000"))
        await hre.network.provider.request({
            method: "hardhat_impersonateAccount",
            params: [governance.address]}
        )
        govSigner = await ethers.getSigner(governance.address);
        await accounts[10].sendTransaction({to:governance.address,value:ethers.utils.parseEther("1.0")}); 
	});

it("init", async function() {

    //require 1: only owner can init governance
    await expect(
        tellor.connect(accounts[5]).init(governance.address),
        "rando account was able to init tellorflex"
    ).to.be.reverted

    //require 3: governance can't be zero address
    await expect(
        tellor.connect(owner).init(0),
        "init was able to set governance to 0 address"
    ).to.be.reverted

    //require 2: can't init twice
    await tellor.connect(owner).init(governance.address)
    await expect(
        tellor.connect(owner).init(governance.address),
        "init was able to be called twice"
    ).to.be.reverted

})
})