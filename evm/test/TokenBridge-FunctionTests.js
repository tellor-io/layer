const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const BN = ethers.BigNumber.from
const abiCoder = new ethers.utils.AbiCoder();
const axios = require('axios');


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
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        
        valTs = await h.getValsetTimestampByIndex(0)
        valParams = await h.getValsetCheckpointParams(valTs)
        valSet = await h.getValset(valParams.timestamp)

        const BlobstreamO = await ethers.getContractFactory("BlobstreamO");
        blobstream = await BlobstreamO.deploy(valParams.powerThreshold, valParams.timestamp, UNBONDING_PERIOD, valParams.checkpoint, guardian.address);
        await blobstream.deployed();

        const Token = await ethers.getContractFactory("TellorPlayground")
        token = await Token.deploy()
        await token.deployed()
        
        const TokenBridge = await ethers.getContractFactory("TokenBridge")
        blocky0 = await h.getBlock()
        tbridge = await TokenBridge.deploy(token.address, blobstream.address)

        // await token.faucet(tbridge.address)
        await token.faucet(accounts[0].address)
    });

    it.only("constructor", async function () {
        assert.equal(await tbridge.token(), token.address)
        assert.equal(await tbridge.bridge(), blobstream.address)
        expect(Number(await tbridge.depositLimitUpdateTime())).to.be.closeTo(Number(blocky0.timestamp), 1)
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10)
        assert.equal(BigInt(await tbridge.currentDepositLimit()), expectedDepositLimit);
    })

    it("withdrawFromLayer", async function () {
        agg = await h.getCurrentAggregateReport(WITHDRAW1_QUERY_ID)

        snapshots = await h.getSnapshotsByReport(WITHDRAW1_QUERY_ID, agg.report.timestamp)
        lastSnapshot = snapshots[snapshots.length - 1]
        attestationData = await h.getAttestationDataBySnapshot(lastSnapshot)
        oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet)

        await h.advanceTime(43200)

        await tbridge.testValset(valSet)
        await tbridge.testSigs(oattests)
        await tbridge.testDepositId(1)

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

    it.only("depositToLayer", async function () {
        depositAmount = h.toWei("1")
        await h.expectThrow(tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)) // not approved
        await token.approve(tbridge.address, h.toWei("1000"))
        await h.expectThrow(tbridge.depositToLayer(0, LAYER_RECIPIENT)) // zero amount
        await h.expectThrow(tbridge.depositToLayer(h.toWei("21"), LAYER_RECIPIENT)) // over limit
        await tbridge.depositToLayer(depositAmount, LAYER_RECIPIENT)
        blocky1 = await h.getBlock()

        tbridgeBal = await token.balanceOf(tbridge.address)
        assert.equal(tbridgeBal.toString(), h.toWei("1"))
        expectedDepositLimit = BigInt(100e18) * BigInt(2) / BigInt(10) - BigInt(depositAmount)
        assert.equal(BigInt(await tbridge.currentDepositLimit()), expectedDepositLimit);
        assert.equal(await tbridge.depositId(), 1)

        depositDetails = await tbridge.deposits(1)
        assert.equal(depositDetails.amount.toString(), depositAmount)
        assert.equal(depositDetails.recipient, LAYER_RECIPIENT)
        assert.equal(depositDetails.sender, accounts[0].address)
        assert.equal(depositDetails.blockHeight, blocky1.number)
    })
})
