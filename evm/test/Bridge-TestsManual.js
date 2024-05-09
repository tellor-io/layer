const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const abiCoder = new ethers.AbiCoder();

describe("BlobstreamO - Manual Function and e2e Tests", function () {

    let bridge, valPower, accounts, validators, powers, initialValAddrs,
        initialPowers, threshold, valCheckpoint, valTimestamp, guardian,
        bridgeCaller;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks

    const ETH_USD_QUERY_ID = "0x83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"

    beforeEach(async function () {
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        initialValAddrs = [accounts[1].getAddress(), accounts[2].getAddress()]
        initialPowers = [1, 2]
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = blocky.timestamp - 2
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, threshold, valTimestamp)

        const Bridge = await ethers.getContractFactory("BlobstreamO");
        bridge = await Bridge.deploy(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint, guardian.address);
        await bridge.deployed();

        const BridgeCaller = await ethers.getContractFactory("BridgeCaller");
        bridgeCaller = await BridgeCaller.deploy(bridge.address);
        await bridgeCaller.deployed();
    });

    it("constructor", async function () {
        assert.equal(await bridge.powerThreshold(), threshold)
        assert.equal(await bridge.validatorTimestamp(), valTimestamp)
        assert.equal(await bridge.unbondingPeriod(), UNBONDING_PERIOD)
        assert.equal(await bridge.lastValidatorSetCheckpoint(), valCheckpoint)
    })

    it("updateValidatorSet", async function () {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
    })

    it("verifyOracleData", async function () {
        queryId = h.hash("myquery")
        value = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp = blocky.timestamp - 2
        aggregatePower = 3
        attestTimestamp = timestamp + 1
        previousTimestamp = 0
        nextTimestamp = 0
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, threshold, valTimestamp)

        dataDigest = await h.getDataDigest(
            queryId,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )

        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(dataDigest))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(dataDigest))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            queryId,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )

        await bridge.verifyOracleData(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray
        )
    })

    it("verifyConsensusOracleData", async function () {
        queryId = h.hash("myquery")
        value = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp = blocky.timestamp - 2
        aggregatePower = 3
        attestTimestamp = timestamp + 1
        previousTimestamp = 0
        nextTimestamp = 0
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, threshold, valTimestamp)

        dataDigest = await h.getDataDigest(
            queryId,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            valCheckpoint,
            attestTimestamp
        )

        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(dataDigest))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(dataDigest))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            queryId,
            value,
            timestamp,
            aggregatePower,
            previousTimestamp,
            nextTimestamp,
            attestTimestamp
        )

        await bridge.verifyConsensusOracleData(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray
        )

        // update validator set
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newThreshold, newValTimestamp)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);

        // verify non-consensus oracle data
        value2 = abiCoder.encode(["uint256"], [3000])
        blocky = await h.getBlock()
        timestamp2 = blocky.timestamp - 2
        aggregatePower2 = 3
        attestTimestamp2 = timestamp2 + 1
        previousTimestamp2 = timestamp
        nextTimestamp2 = 0
        valCheckpoint2 = newValCheckpoint

        dataDigest2 = await h.getDataDigest(
            queryId,
            value2,
            timestamp2,
            aggregatePower2,
            previousTimestamp2,
            nextTimestamp2,
            valCheckpoint2,
            attestTimestamp2
        )

        currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(dataDigest2))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(dataDigest2))
        sig3 = await accounts[3].signMessage(ethers.utils.arrayify(dataDigest2))
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        oracleDataStruct2 = await h.getOracleDataStruct(
            queryId,
            value2,
            timestamp2,
            aggregatePower2,
            previousTimestamp2,
            nextTimestamp2,
            attestTimestamp2
        )
        await bridge.verifyOracleData(
            oracleDataStruct2,
            currentValSetArray2,
            sigStructArray2
        )

        await h.expectThrow(bridge.verifyConsensusOracleData(
            oracleDataStruct2,
            currentValSetArray2,
            sigStructArray2
        ))
    })

    it("guardianResetValidatorSet", async function () {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await h.expectThrow(bridge.connect(guardian).guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint));

        await h.advanceTime(UNBONDING_PERIOD + 1)

        await h.expectThrow(bridge.guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint));
        await bridge.connect(guardian).guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint);
    })

    it("updateValidatorSet twice", async function () {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newThreshold, newValTimestamp)
        newDigest = await h.getEthSignedMessageHash(newValCheckpoint)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);

        newValAddrs2 = [accounts[4].address, accounts[5].address, accounts[6].address, accounts[7].address]
        newPowers2 = [4, 5, 6, 7]
        newThreshold2 = 15
        newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
        blocky = await h.getBlock()
        newValTimestamp2 = blocky.timestamp - 1
        newValCheckpoint2 = h.calculateValCheckpoint(newValAddrs2, newPowers2, newThreshold2, newValTimestamp2)
        newDigest2 = await h.getEthSignedMessageHash(newValCheckpoint2)
        currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig3 = await accounts[3].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        await bridge.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);
    })

    it("alternating validator set updates and verify oracle data", async function () {
        // verify oracle data 
        queryId1 = h.hash("eth-usd")
        value1 = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp1 = blocky.timestamp - 2 // report timestamp
        aggregatePower1 = 3
        attestTimestamp1 = timestamp1 + 1
        previousTimestamp1 = 0
        nextTimestamp1 = 0
        valCheckpoint1 = h.calculateValCheckpoint(initialValAddrs, initialPowers, threshold, valTimestamp)

        dataDigest1 = await h.getDataDigest(
            queryId1,
            value1,
            timestamp1,
            aggregatePower1,
            previousTimestamp1,
            nextTimestamp1,
            valCheckpoint1,
            attestTimestamp1
        )

        currentValSetArray1 = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(dataDigest1))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(dataDigest1))
        sigStructArray1 = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct1 = await h.getOracleDataStruct(
            queryId1,
            value1,
            timestamp1,
            aggregatePower1,
            previousTimestamp1,
            nextTimestamp1,
            attestTimestamp1
        )

        await bridge.verifyOracleData(
            oracleDataStruct1,
            currentValSetArray1,
            sigStructArray1
        )

        // update validator set 
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newThreshold, newValTimestamp)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray1, sigStructArray);

        // verify oracle data
        value2 = abiCoder.encode(["uint256"], [3000])
        blocky = await h.getBlock()
        timestamp2 = blocky.timestamp - 2
        aggregatePower2 = 6
        attestTimestamp2 = timestamp2 + 1
        previousTimestamp2 = timestamp1
        nextTimestamp2 = 0
        valCheckpoint2 = newValCheckpoint

        dataDigest2 = await h.getDataDigest(
            queryId1,
            value2,
            timestamp2,
            aggregatePower2,
            previousTimestamp2,
            nextTimestamp2,
            valCheckpoint2,
            attestTimestamp2
        )

        currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(dataDigest2))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(dataDigest2))
        sig3 = await accounts[3].signMessage(ethers.utils.arrayify(dataDigest2))
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        oracleDataStruct2 = await h.getOracleDataStruct(
            queryId1,
            value2,
            timestamp2,
            aggregatePower2,
            previousTimestamp2,
            nextTimestamp2,
            attestTimestamp2
        )

        await bridge.verifyOracleData(
            oracleDataStruct2,
            currentValSetArray2,
            sigStructArray2
        )

        // update validator set
        newValAddrs2 = [accounts[4].address, accounts[5].address, accounts[6].address, accounts[7].address]
        newPowers2 = [4, 5, 6, 7]
        newThreshold2 = 15
        newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
        blocky = await h.getBlock()
        newValTimestamp2 = blocky.timestamp - 1
        newValCheckpoint2 = h.calculateValCheckpoint(newValAddrs2, newPowers2, newThreshold2, newValTimestamp2)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig3 = await accounts[3].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        await bridge.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);

        // verify oracle data
        value3 = abiCoder.encode(["uint256"], [4000])
        blocky = await h.getBlock()
        timestamp3 = blocky.timestamp - 2
        aggregatePower3 = 22
        attestTimestamp3 = timestamp3 + 1
        previousTimestamp3 = timestamp2
        nextTimestamp3 = 0
        valCheckpoint3 = newValCheckpoint2

        dataDigest3 = await h.getDataDigest(
            queryId1,
            value3,
            timestamp3,
            aggregatePower3,
            previousTimestamp3,
            nextTimestamp3,
            valCheckpoint3,
            attestTimestamp3
        )

        currentValSetArray3 = await h.getValSetStructArray(newValAddrs2, newPowers2)
        sig1 = await accounts[4].signMessage(ethers.utils.arrayify(dataDigest3))
        sig2 = await accounts[5].signMessage(ethers.utils.arrayify(dataDigest3))
        sig3 = await accounts[6].signMessage(ethers.utils.arrayify(dataDigest3))
        sig4 = await accounts[7].signMessage(ethers.utils.arrayify(dataDigest3))
        sigStructArray3 = await h.getSigStructArray([sig1, sig2, sig3, sig4])
        oracleDataStruct3 = await h.getOracleDataStruct(
            queryId1,
            value3,
            timestamp3,
            aggregatePower3,
            previousTimestamp3,
            nextTimestamp3,
            attestTimestamp3
        )

        await bridge.verifyOracleData(
            oracleDataStruct3,
            currentValSetArray3,
            sigStructArray3
        )
    })

    it("update validator set to 100+ validators", async function () {
        nVals = 158
        let wallets = []
        for (i = 0; i < nVals; i++) {
            wallets.push(await ethers.Wallet.createRandom())
        }

        newValAddrs = []
        newValPowers = []
        for (i = 0; i < nVals; i++) {
            newValAddrs.push(wallets[i].address)
            newValPowers.push(1)
        }
        newValHash = await h.calculateValHash(newValAddrs, newValPowers)

        newThreshold = Math.ceil(nVals * 2 / 3)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newValPowers, newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);

        // verify oracle data
        queryId1 = h.hash("eth-usd")
        value1 = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp1 = blocky.timestamp - 2 // report timestamp
        aggregatePower1 = 3
        attestTimestamp1 = timestamp1 + 1
        previousTimestamp1 = 0
        nextTimestamp1 = 0

        dataDigest1 = await h.getDataDigest(
            queryId1,
            value1,
            timestamp1,
            aggregatePower1,
            previousTimestamp1,
            nextTimestamp1,
            newValCheckpoint,
            attestTimestamp1
        )

        currentValSetArray1 = await h.getValSetStructArray(newValAddrs, newValPowers)
        sigs = []
        for (i = 0; i < nVals; i++) {
            sigs.push(await wallets[i].signMessage(ethers.utils.arrayify(dataDigest1)))
        }
        sigStructArray1 = await h.getSigStructArray(sigs)
        oracleDataStruct1 = await h.getOracleDataStruct(
            queryId1,
            value1,
            timestamp1,
            aggregatePower1,
            previousTimestamp1,
            nextTimestamp1,
            attestTimestamp1
        )

        await bridge.verifyOracleData(
            oracleDataStruct1,
            currentValSetArray1,
            sigStructArray1
        )

        await bridgeCaller.verifyAndSaveOracleData(
            oracleDataStruct1,
            currentValSetArray1,
            sigStructArray1
        )

    })
})
