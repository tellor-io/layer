const { ethers} = require("hardhat");
const h = require("../test/helpers/helpers");
var assert = require('assert');

describe("BlobstreamO - Auto Function and e2e Tests", function () {

    let bridge, accounts,initialValAddrs,
        initialPowers, threshold, valCheckpoint, valTimestamp, guardian,
        bridgeCaller;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks
    const ETH_USD_QUERY_ID = "0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"

    beforeEach(async function () {
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        initialValAddrs = [await accounts[1].address, await accounts[2].address]
        initialPowers = [1, 2]
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = blocky.timestamp - 2
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, threshold, valTimestamp)
        bridge = await ethers.deployContract("BlobstreamO", [guardian.address]);
        await bridge.init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint)
        bridgeCaller = await ethers.deployContract("BridgeCaller", [bridge.address]);
    });
    it("constructor", async function () {
        assert.equal(await bridge.powerThreshold(), threshold)
        assert.equal(await bridge.guardian(), await accounts[10].address)
        assert.equal(await bridge.validatorTimestamp(), valTimestamp)
        assert.equal(await bridge.unbondingPeriod(), UNBONDING_PERIOD)
        assert.equal(await bridge.lastValidatorSetCheckpoint(), valCheckpoint)
    })

    it("query layer api, deploy and verify with real params", async function () {
        vts0 = await h.getValsetTimestampByIndex(0)
        vp0 = await h.getValsetCheckpointParams(vts0)
        console.log("deploying bridge...")
        bridge = await ethers.deployContract("BlobstreamO", [guardian.address]);
        await bridge.init(vp0.powerThreshold, vp0.timestamp, UNBONDING_PERIOD, vp0.checkpoint)
        vts1 = await h.getValsetTimestampByIndex(1)
        vp1 = await h.getValsetCheckpointParams(vts1)
        valSet0 = await h.getValset(vp0.timestamp)
        valSet1 = await h.getValset(vp1.timestamp)
        console.log("valSet0: ", valSet0)
        console.log("valSet1: ", valSet1)
        vsigs1 = await h.getValsetSigs(vp1.timestamp, valSet0, vp1.checkpoint)
        console.log("valsetSigs1: ", vsigs1)
        console.log("updating validator set...")
        await bridge.updateValidatorSet(vp1.valsetHash, vp1.powerThreshold, vp1.timestamp, valSet0, vsigs1);
        ethUsdRep0 = await h.getCurrentAggregateReport(ETH_USD_QUERY_ID)
        console.log("ethUsdRep0: ", ethUsdRep0)
        snapshots = await h.getSnapshotsByReport(ETH_USD_QUERY_ID, ethUsdRep0.report.timestamp)
        console.log("snapshots: ", snapshots)
        lastSnapshot = snapshots[snapshots.length - 1]
        attestationData = await h.getAttestationDataBySnapshot(lastSnapshot)
        console.log("attestationData: ", attestationData)
        oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet1)
        if (oattests.length == 0) {
            sleeptime = 2
            console.log("no attestations found, sleeping for ", sleeptime, " seconds...")
            await h.sleep(2)
            oattests = await h.getAttestationsBySnapshot(lastSnapshot, valSet1)
        }
        console.log("oattests: ", oattests)
        console.log("verifying oracle data...")
        await bridge.verifyOracleData(
            attestationData,
            valSet1,
            oattests,
        )
    })

    it("optimistic value", async function () {
        // request new attestations on layer and update PAST_REPORT_TS
        const PAST_REPORT_TS = 1725030526673
        vts0 = await h.getValsetTimestampByIndex(0)
        vp0 = await h.getValsetCheckpointParams(vts0)
        bridge = await ethers.deployContract("BlobstreamO", [guardian.address]);
        await bridge.init(vp0.powerThreshold, vp0.timestamp, UNBONDING_PERIOD, vp0.checkpoint)
        vts1 = await h.getValsetTimestampByIndex(1)
        vp1 = await h.getValsetCheckpointParams(vts1)
        valSet0 = await h.getValset(vp0.timestamp)
        valSet1 = await h.getValset(vp1.timestamp)
        vsigs1 = await h.getValsetSigs(vp1.timestamp, valSet0, vp1.checkpoint)
        await bridge.updateValidatorSet(vp1.valsetHash, vp1.powerThreshold, vp1.timestamp, valSet0, vsigs1);
        // request new attestations
        currentBlock = await h.getBlock()
        currentTime = currentBlock.timestamp
        currentTime = (currentTime - 60) * 1000
        pastReport = await h.getDataBefore(ETH_USD_QUERY_ID, currentTime)
        try {
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
            console.log("oracle data verified")
        } catch (error) {
            console.log("Please request new attestations for the eth/usd past report timestamp %s:", pastReport.timestamp)
            console.log("\n./layerd tx bridge request-attestations {your-address} 83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992 %s --from {your-key-name} --chain-id {chain-id} --keyring-backend {keyring-backend} --keyring-dir {your-keyring-dir} --fees 1000loya --yes", pastReport.timestamp)
            console.log("\nand update PAST_REPORT_TS variable.")
            assert(false)
        }
    })
})
