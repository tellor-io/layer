const { expect } = require("chai");
const { ethers } = require("hardhat");
const h = require("./helpers/evmHelpers");
var assert = require('assert');
const abiCoder = new ethers.AbiCoder();


describe("Blobstream - Function Tests", async function () {

    let blobstream, accounts, guardian, initialPowers, initialValAddrs;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks

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
    });
    it("constructor", async function () {
        assert.equal(await blobstream.powerThreshold(), threshold)
        assert.equal(await blobstream.validatorTimestamp(), valTimestamp)
        assert.equal(await blobstream.unbondingPeriod(), UNBONDING_PERIOD)
        assert.equal(await blobstream.lastValidatorSetCheckpoint(), valCheckpoint)
        assert.equal(await blobstream.guardian(), await guardian.getAddress())
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
        await blobstream.connect(guardian).guardianResetValidatorSet(newThreshold, newValTimestamp, newValCheckpoint);
    })
    it("updateValidatorSet", async function () {
        newValAddrs = [await accounts[1].getAddress(), await accounts[2].getAddress(), await accounts[3].getAddress()]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = await h.calculateValCheckpoint(newValHash,newThreshold, newValTimestamp)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.toBeArray(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.toBeArray(newValCheckpoint))
        sig3 = await accounts[3].signMessage(ethers.toBeArray(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        badSigStructArray = await h.getSigStructArray([sig3, sig2])
        insufStructArray = await h.getSigStructArray([sig1,0])
        await h.expectThrow(blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, insufStructArray));
        //invalid signatures
        await h.expectThrow(blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, badSigStructArray));
        await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
        
        newValAddrs = [accounts[1].address]
        newPowers = [6]
        newThreshold = 5
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValHash, newThreshold, newValTimestamp)
        sig1 = await accounts[1].signMessage(ethers.toBeArray(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.toBeArray(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        //stale validator set
        await h.expectThrow(blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray));
    
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
        sig1 = await accounts[1].signMessage(ethers.getBytes(dataDigest))
        sig2 = await accounts[2].signMessage(ethers.getBytes(dataDigest))
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
            newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
            newPowers = [1, 2, 3]
            newThreshold = 4
            newValHash = await h.calculateValHash(newValAddrs, newPowers)
            blocky = await h.getBlock()
            newValTimestamp = blocky.timestamp - 1
            newValCheckpoint = h.calculateValCheckpoint(newValHash, newThreshold, newValTimestamp)
            currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
            sig1 = await accounts[1].signMessage(ethers.getBytes(newValCheckpoint))
            sig2 = await accounts[2].signMessage(ethers.getBytes(newValCheckpoint))
            sigStructArray = await h.getSigStructArray([sig1, sig2])
            await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
            newValAddrs2 = [accounts[4].address, accounts[5].address, accounts[6].address, accounts[7].address]
            newPowers2 = [4, 5, 6, 7]
            newThreshold2 = 15
            newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
            blocky = await h.getBlock()
            newValTimestamp2 = blocky.timestamp - 1
            newValCheckpoint2 = h.calculateValCheckpoint(newValHash2, newThreshold2, newValTimestamp2)
            currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
            sig1 = await accounts[1].signMessage(ethers.getBytes(newValCheckpoint2))
            sig2 = await accounts[2].signMessage(ethers.getBytes(newValCheckpoint2))
            sig3 = await accounts[3].signMessage(ethers.getBytes(newValCheckpoint2))
            sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
            await blobstream.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);
            
        })
        it("alternating validator set updates and verify oracle data", async function () {
            queryId1 = h.hash("eth-usd")
            value1 = abiCoder.encode(["uint256"], [2000])
            blocky = await h.getBlock()
            timestamp1 = blocky.timestamp - 2 // report timestamp
            aggregatePower1 = 3
            attestTimestamp1 = timestamp1 + 1
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
            sig1 = await accounts[1].signMessage(ethers.getBytes(dataDigest1))
            sig2 = await accounts[2].signMessage(ethers.getBytes(dataDigest1))
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
            newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
            newPowers = [1, 2, 3]
            newThreshold = 4
            newValHash = await h.calculateValHash(newValAddrs, newPowers)
            blocky = await h.getBlock()
            newValTimestamp = blocky.timestamp - 1
            newValCheckpoint = h.calculateValCheckpoint(newValHash,newThreshold, newValTimestamp)
            sig1 = await accounts[1].signMessage(ethers.getBytes(newValCheckpoint))
            sig2 = await accounts[2].signMessage(ethers.getBytes(newValCheckpoint))
            sigStructArray = await h.getSigStructArray([sig1, sig2])
            await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray1, sigStructArray);
    
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
            sig1 = await accounts[1].signMessage(ethers.getBytes(dataDigest2))
            sig2 = await accounts[2].signMessage(ethers.getBytes(dataDigest2))
            sig3 = await accounts[3].signMessage(ethers.getBytes(dataDigest2))
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
            newValAddrs2 = [accounts[4].address, accounts[5].address, accounts[6].address, accounts[7].address]
            newPowers2 = [4, 5, 6, 7]
            newThreshold2 = 15
            newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
            blocky = await h.getBlock()
            newValTimestamp2 = blocky.timestamp - 1
            newValCheckpoint2 = h.calculateValCheckpoint(newValHash2, newThreshold2, newValTimestamp2)
            sig1 = await accounts[1].signMessage(ethers.getBytes(newValCheckpoint2))
            sig2 = await accounts[2].signMessage(ethers.getBytes(newValCheckpoint2))
            sig3 = await accounts[3].signMessage(ethers.getBytes(newValCheckpoint2))
            sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
            await blobstream.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);
    
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
            sig1 = await accounts[4].signMessage(ethers.getBytes(dataDigest3))
            sig2 = await accounts[5].signMessage(ethers.getBytes(dataDigest3))
            sig3 = await accounts[6].signMessage(ethers.getBytes(dataDigest3))
            sig4 = await accounts[7].signMessage(ethers.getBytes(dataDigest3))
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
            newValCheckpoint = h.calculateValCheckpoint(newValHash, newThreshold, newValTimestamp)
            currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
            sig1 = await accounts[1].signMessage(ethers.getBytes(newValCheckpoint))
            sig2 = await accounts[2].signMessage(ethers.getBytes(newValCheckpoint))
            sigStructArray = await h.getSigStructArray([sig1, sig2])
            await blobstream.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
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
                sigs.push(await wallets[i].signMessage(ethers.getBytes(dataDigest1)))
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
        })
    })
