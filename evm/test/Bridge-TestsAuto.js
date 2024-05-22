const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const abiCoder = new ethers.AbiCoder();

describe("BlobstreamO - Auto Function and e2e Tests", function () {

    let bridge, valPower, accounts, validators, powers, initialValAddrs,
        initialPowers, threshold, valCheckpoint, valTimestamp, guardian,
        bridgeCaller;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks
    const ETH_USD_QUERY_ID = "0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
    const PAST_REPORT_TS = 0

    beforeEach(async function () {
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        initialValAddrs = [await accounts[1].getAddress(), await accounts[2].getAddress()]
        initialPowers = [1, 2]
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = blocky.timestamp - 2
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, threshold, valTimestamp)

        bridge = await ethers.deployContract("BlobstreamO", [threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint, guardian.getAddress()]);
        bridgeCaller = await ethers.deployContract("BridgeCaller", [bridge.getAddress()]);
    });

    it("constructor", async function () {
        assert.equal(await bridge.powerThreshold(), threshold)
        assert.equal(await bridge.validatorTimestamp(), valTimestamp)
        assert.equal(await bridge.unbondingPeriod(), UNBONDING_PERIOD)
        assert.equal(await bridge.lastValidatorSetCheckpoint(), valCheckpoint)
    })

    it("query layer api, deploy and verify with real params", async function () {
        vts0 = await h.getValsetTimestampByIndex(0)
        vp0 = await h.getValsetCheckpointParams(vts0)

        bridge = await ethers.deployContract("BlobstreamO", [vp0.powerThreshold, vp0.timestamp, UNBONDING_PERIOD, vp0.checkpoint, await guardian.getAddress()]);

        vts1 = await h.getValsetTimestampByIndex(1)
        vp1 = await h.getValsetCheckpointParams(vts1)
        valSet0 = await h.getValset(vp0.timestamp)
        valSet1 = await h.getValset(vp1.timestamp)

        vsigs1 = await h.getValsetSigs(vp1.timestamp, valSet0, vp1.checkpoint)

        await bridge.updateValidatorSet(vp1.valsetHash, vp1.powerThreshold, vp1.timestamp, valSet0, vsigs1);

        ethUsdRep0 = await h.getCurrentAggregateReport(ETH_USD_QUERY_ID)
        snapshots = await h.getSnapshotsByReport(ETH_USD_QUERY_ID, ethUsdRep0.report.timestamp)
        lastSnapshot = snapshots[snapshots.length - 1]
        attestationData = await h.getAttestationDataBySnapshot(lastSnapshot)

        oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet1)
        if (oattests.length == 0) {
            sleeptime = 2
            console.log("no attestations found, sleeping for ", sleeptime, " seconds...")
            await h.sleep(2)
            oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet1)
        }

        await bridge.verifyOracleData(
            attestationData,
            valSet1,
            oattests,
        )
        // log instructions for "optimistic value" test
        queryIdWithout0x = ETH_USD_QUERY_ID.slice(2)
        if (PAST_REPORT_TS == 0) {
            console.log("to test 'optimistic value', run:")
            console.log("./layerd tx bridge request-attestations %s %s --from charlie --chain-id layer --home ~/.layer/alice", queryIdWithout0x, attestationData.report.timestamp)
            console.log("update 'PAST_REPORT_TS' to %s", attestationData.report.timestamp)
        }
    })

    it("optimistic value", async function () {
        // request new attestations on layer and update PAST_REPORT_TS above

        if (PAST_REPORT_TS == 0) {
            assert.fail("PAST_REPORT_TS is 0. Please request new attestations and update PAST_REPORT_TS above")
        }
        
        vts0 = await h.getValsetTimestampByIndex(0)
        vp0 = await h.getValsetCheckpointParams(vts0)

        bridge = await ethers.deployContract("BlobstreamO", [vp0.powerThreshold, vp0.timestamp, UNBONDING_PERIOD, vp0.checkpoint, guardian.address]);

        vts1 = await h.getValsetTimestampByIndex(1)
        vp1 = await h.getValsetCheckpointParams(vts1)
        valSet0 = await h.getValset(vp0.timestamp)
        valSet1 = await h.getValset(vp1.timestamp)

        vsigs1 = await h.getValsetSigs(vp1.timestamp, valSet0, vp1.checkpoint)
        
        await bridge.updateValidatorSet(vp1.valsetHash, vp1.powerThreshold, vp1.timestamp, valSet0, vsigs1);


        // request new attestations
        currentBlock = await h.getBlock()
        currentTime = currentBlock.timestamp
        currentTime = currentTime - 10
        pastReport = await h.getDataBefore(ETH_USD_QUERY_ID, currentTime)

        snapshots = await h.getSnapshotsByReport(ETH_USD_QUERY_ID, PAST_REPORT_TS)
        lastSnapshot = snapshots[1]
        attestationData = await h.getAttestationDataBySnapshot(lastSnapshot)

        oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet1)
        if (oattests.length == 0) {
            sleeptime = 2
            console.log("no attestations found, sleeping for ", sleeptime, " seconds...")
            await h.sleep(2)
            oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet1)
        }

        assert(await bridge.verifyOracleData(
            attestationData,
            valSet1,
            oattests,
        ))
    })
})
