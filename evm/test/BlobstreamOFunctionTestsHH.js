const { expect } = require("chai");
const { ethers } = require("hardhat");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const abiCoder = new ethers.utils.AbiCoder();
// const Web3 = require('web3');
// const web3 = require("@nomiclabs/hardhat-web3");
const hre = require("hardhat");
const EC = require('elliptic').ec;
const ec = new EC('secp256k1');


describe("Blobstream - Function Tests", async function () {

    let blobstream, accounts, guardian, initialPowers, initialValAddrs;
    let val1, val2, val3;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks

    beforeEach(async function () {
        // init accounts
        accounts = await ethers.getSigners();
        guardian = accounts[10]
        val1 = ethers.Wallet.createRandom()
        val2 = ethers.Wallet.createRandom()
        val3 = ethers.Wallet.createRandom()
        val4 = ethers.Wallet.createRandom()
        val5 = ethers.Wallet.createRandom()
        val6 = ethers.Wallet.createRandom()
        val7 = ethers.Wallet.createRandom()
        val8 = ethers.Wallet.createRandom()
        val9 = ethers.Wallet.createRandom()
        val10 = ethers.Wallet.createRandom()
        initialValAddrs = [val1.address, val2.address]
        initialPowers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = (blocky.timestamp - 2) * 1000
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
        // deploy contracts
        blobstream = await ethers.deployContract(
            "BlobstreamO", [
            guardian.address
        ]
        )
        await blobstream.init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint)
    });

    it("constructor", async function () {
        assert.equal(await blobstream.guardian(), await guardian.address)
    })
    it("init", async function() {
        // deploy a new blobstream contract, same inputs as above
        blobstream2 = await ethers.deployContract("BlobstreamO", [
            guardian.address
        ])
        await h.expectThrow(blobstream2.connect(accounts[1]).init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint))
        await blobstream2.init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint)
        assert.equal(await blobstream2.powerThreshold(), threshold)
        assert.equal(await blobstream2.validatorTimestamp(), valTimestamp)
        assert.equal(await blobstream2.unbondingPeriod(), UNBONDING_PERIOD)
        assert.equal(await blobstream2.lastValidatorSetCheckpoint(), valCheckpoint)
        await h.expectThrow(blobstream2.init(threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint))
    })
    it("guardianResetValidatorSet", async function () {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValHash,newThreshold, newValTimestamp)
        await h.expectThrow(blobstream.connect(guardian).guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint));
        await h.advanceTime(UNBONDING_PERIOD + 1)
        await h.expectThrow(blobstream.guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint));//not guardian
        oldValTimestamp = await blobstream.validatorTimestamp()
        await h.expectThrow(blobstream.connect(guardian).guardianResetValidatorSet(newThreshold, oldValTimestamp, newValCheckpoint));
        blocky = await h.getBlock()
        newValTimestamp = (blocky.timestamp - 1) * 1000
        await blobstream.connect(guardian).guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint);
        assert.equal(await blobstream.validatorTimestamp(), newValTimestamp)
        assert.equal(await blobstream.lastValidatorSetCheckpoint(), newValCheckpoint)
        assert.equal(await blobstream.powerThreshold(), newThreshold)
    })
    it("updateValidatorSet", async function () {
        newValAddrs = [val1.address, val2.address, val3.address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = (blocky.timestamp - 1) * 1000
        newValCheckpoint = await h.calculateValCheckpoint(newValHash,newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await h.layerSign(newValCheckpoint, val1.privateKey)
        sig2 = await h.layerSign(newValCheckpoint, val2.privateKey)
        sig3 = await h.layerSign(newValCheckpoint, val3.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        badSigStructArray = await h.getSigStructArray([sig3, sig2])
        insufStructArray = await h.getSigStructArray([sig1,{v: 0, r: 0, s: 0}])
        await h.expectThrow(blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, insufStructArray));
        //invalid signatures
        await h.expectThrow(blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, badSigStructArray));
        await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
        
        newValAddrs = [accounts[1].address]
        newPowers = [6]
        newThreshold = 5
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = (blocky.timestamp - 1) * 1000
        newValCheckpoint = h.calculateValCheckpoint(newValHash, newThreshold, newValTimestamp)
        sig1 = await h.layerSign(newValCheckpoint, val1.privateKey)
        sig2 = await h.layerSign(newValCheckpoint, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        //stale validator set
        await h.expectThrow(blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray));
    
    })
    it("verifyOracleData", async function () {
        queryId = h.hash("myquery")
        value = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp = (blocky.timestamp - 2) * 1000
        aggregatePower = 3
        attestTimestamp = timestamp + 1000
        previousTimestamp = 0
        nextTimestamp = 0
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint = await h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
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
        sig1 = await h.layerSign(dataDigest, val1.privateKey)
        sig2 = await h.layerSign(dataDigest, val2.privateKey)
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
        await blobstream.verifyOracleData(
            oracleDataStruct,
            currentValSetArray,
            sigStructArray
        )
    })
    //more complex    
    it("updateValidatorSet twice", async function () {
        // update val set 1
        newValAddrs = [val1.address, val2.address, val3.address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = (blocky.timestamp - 1) * 1000
        newValCheckpoint = h.calculateValCheckpoint(newValHash, newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await h.layerSign(newValCheckpoint, val1.privateKey)
        sig2 = await h.layerSign(newValCheckpoint, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);

        // update val set 2
        newValAddrs2 = [val4.address, val5.address, val6.address, val7.address]
        newPowers2 = [4, 5, 6, 7]
        newThreshold2 = 15
        newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
        blocky = await h.getBlock()
        newValTimestamp2 = (blocky.timestamp - 1) * 1000
        newValCheckpoint2 = h.calculateValCheckpoint(newValHash2, newThreshold2, newValTimestamp2)
        currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
        sig1 = await h.layerSign(newValCheckpoint2, val1.privateKey)
        sig2 = await h.layerSign(newValCheckpoint2, val2.privateKey)
        sig3 = await h.layerSign(newValCheckpoint2, val3.privateKey)
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        await blobstream.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);
        
    })
    it("alternating validator set updates and verify oracle data", async function () {
        queryId1 = h.hash("eth-usd")
        value1 = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp1 = (blocky.timestamp - 2) * 1000 // report timestamp
        aggregatePower1 = 3
        attestTimestamp1 = timestamp1 + 1000
        previousTimestamp1 = 0
        nextTimestamp1 = 0
        newValHash = await h.calculateValHash(initialValAddrs, initialPowers)
        valCheckpoint1 = h.calculateValCheckpoint(newValHash, threshold, valTimestamp)
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
        sig1 = await h.layerSign(dataDigest1, val1.privateKey)
        sig2 = await h.layerSign(dataDigest1, val2.privateKey)
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
        await blobstream.verifyOracleData(
            oracleDataStruct1,
            currentValSetArray1,
            sigStructArray1
        )
        // update validator set 
        newValAddrs = [val1.address, val2.address, val3.address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = (blocky.timestamp - 1) * 1000
        newValCheckpoint = h.calculateValCheckpoint(newValHash,newThreshold, newValTimestamp)
        sig1 = h.layerSign(newValCheckpoint, val1.privateKey)
        sig2 = h.layerSign(newValCheckpoint, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray1, sigStructArray);

        // verify oracle data
        value2 = abiCoder.encode(["uint256"], [3000])
        blocky = await h.getBlock()
        timestamp2 = (blocky.timestamp - 2) * 1000
        aggregatePower2 = 6
        attestTimestamp2 = timestamp2 + 1000
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
        sig1 = await h.layerSign(dataDigest2, val1.privateKey)
        sig2 = await h.layerSign(dataDigest2, val2.privateKey)
        sig3 = await h.layerSign(dataDigest2, val3.privateKey)
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
        await blobstream.verifyOracleData(
            oracleDataStruct2,
            currentValSetArray2,
            sigStructArray2
        )
        // update validator set
        newValAddrs2 = [val4.address, val5.address, val6.address, val7.address]
        newPowers2 = [4, 5, 6, 7]
        newThreshold2 = 15
        newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
        blocky = await h.getBlock()
        newValTimestamp2 = (blocky.timestamp - 1) * 1000
        newValCheckpoint2 = h.calculateValCheckpoint(newValHash2, newThreshold2, newValTimestamp2)
        sig1 = await h.layerSign(newValCheckpoint2, val1.privateKey)
        sig2 = await h.layerSign(newValCheckpoint2, val2.privateKey)
        sig3 = await h.layerSign(newValCheckpoint2, val3.privateKey)
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        await blobstream.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);

        // verify oracle data
        value3 = abiCoder.encode(["uint256"], [4000])
        blocky = await h.getBlock()
        timestamp3 = (blocky.timestamp - 2) * 1000
        aggregatePower3 = 22
        attestTimestamp3 = timestamp3 + 1000
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
        sig1 = await h.layerSign(dataDigest3, val4.privateKey)
        sig2 = await h.layerSign(dataDigest3, val5.privateKey)
        sig3 = await h.layerSign(dataDigest3, val6.privateKey)
        sig4 = await h.layerSign(dataDigest3, val7.privateKey)
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
        await blobstream.verifyOracleData(
            oracleDataStruct3,
            currentValSetArray3,
            sigStructArray3
        )
    })
    it("update validator set to 100+ validators", async function () {
        // update val set to 100+ validators
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
        newValTimestamp = (blocky.timestamp - 1) * 1000
        newValCheckpoint = h.calculateValCheckpoint(newValHash, newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = h.layerSign(newValCheckpoint, val1.privateKey)
        sig2 = h.layerSign(newValCheckpoint, val2.privateKey)
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
        
        // verify oracle data
        queryId1 = h.hash("eth-usd")
        value1 = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp1 = (blocky.timestamp - 2) * 1000 // report timestamp
        aggregatePower1 = 3
        attestTimestamp1 = timestamp1 + 1000
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
            sigs.push(h.layerSign(dataDigest1, wallets[i].privateKey))
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
        await blobstream.verifyOracleData(
            oracleDataStruct1,
            currentValSetArray1,
            sigStructArray1
        )

        // update again 
        newValAddrs2 = [val4.address, val5.address, val6.address, val7.address]
        newPowers2 = [4, 5, 6, 7]
        newThreshold2 = 15
        newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
        blocky = await h.getBlock()
        newValTimestamp2 = (blocky.timestamp - 1) * 1000
        newValCheckpoint2 = h.calculateValCheckpoint(newValHash2, newThreshold2, newValTimestamp2)
        sigs2 = []
        for (i = 0; i < nVals; i++) {
            sigs2.push(h.layerSign(newValCheckpoint2, wallets[i].privateKey))
        }
        sigStructArray2 = await h.getSigStructArray(sigs2)
        await blobstream.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray1, sigStructArray2);
    })
    })
