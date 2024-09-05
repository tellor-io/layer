const { expect } = require("chai");
const { ethers } = require("hardhat");
const h = require("../test/helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const abiCoder = new ethers.utils.AbiCoder();


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
        // get inital layer valset params
        valTs = await h.getValsetTimestampByIndex(0)
        valParams = await h.getValsetCheckpointParams(valTs)
        valSet = await h.getValset(valParams.timestamp)
        // deploy contracts
        blobstream = await ethers.deployContract("BlobstreamO", [guardian.address])
        await blobstream.init(valParams.powerThreshold, valParams.timestamp, UNBONDING_PERIOD, valParams.checkpoint)
        token = await ethers.deployContract("TellorPlayground")
        oldOracle = await ethers.deployContract("TellorPlayground")
        tbridge = await ethers.deployContract("TokenBridge", [token.address, blobstream.address, oldOracle.address])
        blocky0 = await h.getBlock()
        // fund accounts
        await token.faucet(accounts[0].address)
        await token.faucet(accounts[10].address)
        await token.connect(accounts[10]).transfer(tbridge.address, h.toWei("100"))
    });

    it("constructor", async function () {
        assert.equal(await tbridge.token(), await token.address)
        assert.equal(await tbridge.bridge(), await blobstream.address)
        assert.equal(await tbridge.tellorFlex(), await oldOracle.address)
    })

    it.skip("withdrawFromLayer", async function () {
        agg = await h.getCurrentAggregateReport(WITHDRAW1_QUERY_ID)

        snapshots = await h.getSnapshotsByReport(WITHDRAW1_QUERY_ID, agg.report.timestamp)
        lastSnapshot = snapshots[snapshots.length - 1]
        attestationData = await h.getAttestationDataBySnapshot(lastSnapshot)
        oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet)

        await h.advanceTime(43200)

        await tbridge.withdrawFromLayer(
            attestationData,
            valSet,
            oattests,
            1,
        )

        recipientBal = await token.balanceOf(EVM_RECIPIENT)
        expectedBal = 100e12 // 100 loya
        assert.equal(recipientBal.toString(), expectedBal)
    })

    it("depositToLayer", async function () {
        depositAmount = h.toWei("1")
        assert.equal(await token.balanceOf(await accounts[0].address), h.toWei("1000"))
        await h.expectThrow(tbridge.depositToLayer(depositAmount, 0, LAYER_RECIPIENT)) // not approved
        await token.approve(await tbridge.address, h.toWei("1000"))
        await h.expectThrow(tbridge.depositToLayer(0, 0, LAYER_RECIPIENT)) // zero amount
        await h.expectThrow(tbridge.depositToLayer(h.toWei("21"), 0, LAYER_RECIPIENT)) // over limit
        await tbridge.depositToLayer(depositAmount, 0, LAYER_RECIPIENT)
        blocky1 = await h.getBlock()

        tbridgeBal = await token.balanceOf(await tbridge.address)
        assert.equal(tbridgeBal.toString(), h.toWei("101"))
        userBal = await token.balanceOf(await accounts[0].address)
        assert.equal(userBal.toString(), h.toWei("999"))
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        assert.equal(BigInt(await tbridge.depositLimitRecord()), expectedDepositLimit);
        assert.equal(BigInt(await tbridge.depositLimit()), expectedDepositLimit);
        assert.equal(await tbridge.depositId(), 1)

        depositDetails = await tbridge.deposits(1)
        assert.equal(depositDetails.amount.toString(), depositAmount)
        assert.equal(depositDetails.recipient, LAYER_RECIPIENT)
        assert.equal(depositDetails.sender, await accounts[0].address)
        assert.equal(depositDetails.blockHeight, blocky1.number)

        assert.equal(await tbridge.depositId(), 1)

        await h.advanceTime(43201)
        expectedDepositLimit2 = (BigInt(100e18) + BigInt(depositAmount)) * BigInt(2) / BigInt(10)
        assert.equal(BigInt(await tbridge.depositLimit()), expectedDepositLimit2);
    })

    it("depositLimit", async function () {
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10)
        assert.equal(BigInt(await tbridge.depositLimit()), expectedDepositLimit);
        await token.approve(await tbridge.address, h.toWei("1000"))
        depositAmount = h.toWei("2")
        await tbridge.depositToLayer(depositAmount, 0, LAYER_RECIPIENT)
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        assert.equal(BigInt(await tbridge.depositLimit()), expectedDepositLimit);
        await h.advanceTime(43201)
        expectedDepositLimit2 = (BigInt(100e18) + BigInt(depositAmount)) / BigInt(5)
        assert.equal(BigInt(await tbridge.depositLimit()), expectedDepositLimit2);
    })
})
