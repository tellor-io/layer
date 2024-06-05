const { expect } = require("chai");
const { ethers } = require("hardhat");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const abiCoder = new ethers.AbiCoder();


describe("TokenBridge - Function Tests", async function () {

    let blobstream, accounts, guardian, tbridge, token, blocky0,
        valTs, valParams, valSet;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks
    const WITHDRAW1_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, 1])
    const WITHDRAW1_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW1_QUERY_DATA_ARGS])
    const WITHDRAW1_QUERY_ID = h.hash(WITHDRAW1_QUERY_DATA)
    const EVM_RECIPIENT = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"
    const LAYER_RECIPIENT = "tellor1zy50vdk8fdae0var2ryjhj2ysxtcm8dp2qtckd"

    beforeEach(async function () {
        // init accounts
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        val1 = await accounts[1].getAddress();
        val2 = await accounts[2].getAddress()
        initialValAddrs = [val1,val2]
        initialPowers = [1, 2]
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = blocky.timestamp - 2
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        // deploy contracts
        blobstream = await ethers.deployContract(
            "BlobstreamO", [
            threshold,
            valTimestamp,
            UNBONDING_PERIOD,
            valCheckpoint,
            guardian.getAddress()
        ]
        )
        token = await ethers.deployContract("TellorPlayground")
        oldOracle = await ethers.deployContract("TellorPlayground")
        tbridge = await ethers.deployContract("TestTokenBridge", [token.getAddress(),blobstream.getAddress(), oldOracle.getAddress()])
        blocky0 = await h.getBlock()
        // fund accounts
        await token.faucet(accounts[0].getAddress())
    });

    it("constructor", async function () {
        assert.equal(await tbridge.token(), await token.getAddress())
        assert.equal(await tbridge.bridge(), await blobstream.getAddress())
        expect(Number(await tbridge.depositLimitUpdateTime())).to.be.closeTo(Number(blocky0.timestamp), 1)
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
    })
    it("withdrawFromLayer", async function () {
        depositAmount = h.toWei("20")
        await h.expectThrow(tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)) // not approved
        await token.approve(await tbridge.getAddress(), h.toWei("100"))
        await tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)
        value = h.getWithdrawValue(EVM_RECIPIENT,LAYER_RECIPIENT,20)
        blocky = await h.getBlock()
        timestamp = blocky.timestamp - 2
        aggregatePower = 3
        attestTimestamp = timestamp + 1
        previousTimestamp = 0
        nextTimestamp = 0
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = await h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        dataDigest = await h.getDataDigest(
            WITHDRAW1_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.getBytes(dataDigest))
        sig2 = await accounts[2].signMessage(ethers.getBytes(dataDigest))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            WITHDRAW1_QUERY_ID,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )
        await h.advanceTime(43200)
        await tbridge.withdrawFromLayer(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray,
            1,
        )
        recipientBal = await token.balanceOf(EVM_RECIPIENT)
        expectedBal = 20e12 // 20 loya
        assert.equal(recipientBal.toString(), expectedBal)
    })
    it("depositToLayer", async function () {
        depositAmount = h.toWei("1")
        assert.equal(await token.balanceOf(await accounts[0].getAddress()), h.toWei("1000"))
        await h.expectThrow(tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)) // not approved
        await token.approve(await tbridge.getAddress(), h.toWei("900"))
        await h.expectThrow(tbridge.depositToLayer(0, LAYER_RECIPIENT)) // zero amount
        await h.expectThrow(tbridge.depositToLayer(h.toWei("21"), LAYER_RECIPIENT)) // over limit
        await tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)
        blocky1 = await h.getBlock()
        tbridgeBal = await token.balanceOf(await tbridge.getAddress())
        assert.equal(tbridgeBal.toString(), h.toWei("1"))
        userBal = await token.balanceOf(await accounts[0].getAddress())
        assert.equal(userBal.toString(), h.toWei("999"))
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        await tbridge.depositLimit()
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        assert.equal(await tbridge.depositId(), 1)
        depositDetails = await tbridge.deposits(1)
        assert.equal(depositDetails.amount.toString(), depositAmount)
        assert.equal(depositDetails.recipient, LAYER_RECIPIENT)
        assert.equal(depositDetails.sender, await accounts[0].getAddress())
        assert.equal(depositDetails.blockHeight, blocky1.number)
        assert.equal(await tbridge.depositId(), 1)
        await h.advanceTime(43200)
        expectedDepositLimit2 = (BigInt(100e18) + BigInt(depositAmount)) * BigInt(2) / BigInt(10)
        await tbridge.depositLimit()
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit2);
    })
    it("depositLimit", async function () {
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10)
        await tbridge.depositLimit()
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        await token.approve(await tbridge.getAddress(), h.toWei("900"))
        depositAmount = h.toWei("2")
        await tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        await tbridge.depositLimit()
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        await h.advanceTime(43200)
        expectedDepositLimit2 = (BigInt(100e18) + BigInt(depositAmount)) / BigInt(5)
        await tbridge.depositLimit()
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit2);
    })
})
