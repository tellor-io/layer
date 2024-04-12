const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const BN = ethers.BigNumber.from
const abiCoder = new ethers.utils.AbiCoder();
const axios = require('axios');


describe("TokenBridge - Function Tests", function () {

    let blobstream, accounts, guardian, tbridge, token,
        valTs, valParams, valSet;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks

    const WITHDRAW1_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, 1])
    const WITHDRAW1_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW1_QUERY_DATA_ARGS])
    const WITHDRAW1_QUERY_ID = h.hash(WITHDRAW1_QUERY_DATA)

    const RECIPIENT = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0"

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
        tbridge = await TokenBridge.deploy(token.address, blobstream.address)

        await token.faucet(tbridge.address)
    });

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

        recipientBal = await token.balanceOf(RECIPIENT)
        expectedBal = 100e12 // 100 loya
        assert.equal(recipientBal.toString(), expectedBal)
    })
})
