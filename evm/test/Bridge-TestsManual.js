const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const BN = ethers.BigNumber.from
const abiCoder = new ethers.utils.AbiCoder();

describe("BlobstreamO - Manual Function and e2e Tests", function () {

    let bridge, valPower, accounts, validators, powers, initialValAddrs, 
        initialPowers, nonce, threshold, valCheckpoint, valTimestamp;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks

    beforeEach(async function () {
        accounts = await ethers.getSigners();
        initialValAddrs = [accounts[1].address, accounts[2].address]
        initialPowers = [1, 2]
        nonce = 1
        threshold = 2
        blocky = await h.getBlock()
        valTimestamp = blocky.timestamp - 2
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, nonce, threshold, valTimestamp)

        const Bridge = await ethers.getContractFactory("BlobstreamO");
        bridge = await Bridge.deploy(nonce, threshold, valTimestamp, UNBONDING_PERIOD, valCheckpoint);
        await bridge.deployed();
    });

    it("constructor", async function () {
        assert.equal(await bridge.validatorNonce(), nonce)
        assert.equal(await bridge.powerThreshold(), threshold)
        assert.equal(await bridge.validatorTimestamp(), valTimestamp)
        assert.equal(await bridge.unbondingPeriod(), UNBONDING_PERIOD)
        assert.equal(await bridge.lastValidatorSetCheckpoint(), valCheckpoint)
    })

    it("updateValidatorSet", async function() {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        newNonce = nonce + 1
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newNonce, newThreshold, newValTimestamp)
        newDigest = await h.getEthSignedMessageHash(newValCheckpoint)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);
    })

    it("verifyOracleData", async function() {
        queryId = h.hash("myquery")
        value = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp = blocky.timestamp - 2
        aggregatePower = 3
        attestTimestamp = timestamp + 1
        previousTimestamp = 0
        nextTimestamp = 0
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, nonce, threshold, valTimestamp)

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

    it("updateValidatorSet twice", async function() {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        newNonce = nonce + 1
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newNonce, newThreshold, newValTimestamp)
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
        newNonce2 = newNonce + 1
        blocky = await h.getBlock()
        newValTimestamp2 = blocky.timestamp - 1
        newValCheckpoint2 = h.calculateValCheckpoint(newValAddrs2, newPowers2, newNonce2, newThreshold2, newValTimestamp2)
        newDigest2 = await h.getEthSignedMessageHash(newValCheckpoint2)
        currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig3 = await accounts[3].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        await bridge.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);
    })   
    
    it("updateValidatorSet twice and verifyOracleData", async function() {
        // update validator set 1
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        newNonce = nonce + 1
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newNonce, newThreshold, newValTimestamp)
        newDigest = await h.getEthSignedMessageHash(newValCheckpoint)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, newValTimestamp, currentValSetArray, sigStructArray);

        // update validator set 2
        const newValAddrs2 = [accounts[4].address, accounts[5].address, accounts[6].address, accounts[7].address]
        const newPowers2 = [4, 5, 6, 7]
        const newThreshold2 = 15
        const newValHash2 = await h.calculateValHash(newValAddrs2, newPowers2)
        const newNonce2 = newNonce + 1
        blocky = await h.getBlock()
        const newValTimestamp2 = blocky.timestamp - 1
        const newValCheckpoint2 = h.calculateValCheckpoint(newValAddrs2, newPowers2, newNonce2, newThreshold2, newValTimestamp2)
        const newDigest2 = await h.getEthSignedMessageHash(newValCheckpoint2)
        currentValSetArray2 = await h.getValSetStructArray(newValAddrs, newPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sig3 = await accounts[3].signMessage(ethers.utils.arrayify(newValCheckpoint2))
        sigStructArray2 = await h.getSigStructArray([sig1, sig2, sig3])
        await bridge.updateValidatorSet(newValHash2, newThreshold2, newValTimestamp2, currentValSetArray2, sigStructArray2);
        lastValSetCheckpoint = await bridge.lastValidatorSetCheckpoint()
        assert.equal(lastValSetCheckpoint, newValCheckpoint2)

        // verify oracle data
        queryId = h.hash("myquery")
        value = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp = blocky.timestamp - 2
        aggregatePower = 3
        attestTimestamp = timestamp + 1
        
        dataDigest = await h.getDataDigest(
            queryId, 
            value, 
            timestamp, 
            aggregatePower, 
            newNonce2, 
            newThreshold2, 
            newValHash2, 
            attestTimestamp
        )

        sig1 = await accounts[4].signMessage(ethers.utils.arrayify(dataDigest))
        sig2 = await accounts[5].signMessage(ethers.utils.arrayify(dataDigest))
        sig3 = await accounts[6].signMessage(ethers.utils.arrayify(dataDigest))
        sig4 = await accounts[7].signMessage(ethers.utils.arrayify(dataDigest))

        sigStructArray = await h.getSigStructArray([sig1, sig2, sig3, sig4])
        oracleDataStruct = await h.getOracleDataStruct(
            queryId, 
            value, 
            timestamp, 
            aggregatePower, 
            newNonce2, 
            newThreshold2, 
            newValHash2, 
            attestTimestamp
        )
        console.log("newValHash2", newValHash2)
        await bridge.verifyOracleData(
            oracleDataStruct, 
            currentValSetArray2, 
            sigStructArray2
        )
    })

    it("alternating validator set updates and verify oracle data", async function() {
        // verify oracle data 
        queryId1 = h.hash("eth-usd")
        value1 = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp1 = blocky.timestamp - 2
        aggregatePower1 = 3
        attestTimestamp1 = timestamp1 + 1
        previousTimestamp1 = 0
        nextTimestamp1 = 0
        valCheckpoint1 = h.calculateValCheckpoint(initialValAddrs, initialPowers, nonce, threshold, valTimestamp)

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
        newNonce = nonce + 1
        blocky = await h.getBlock()
        newValTimestamp = blocky.timestamp - 1
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, newNonce, newThreshold, newValTimestamp)
        newDigest = await h.getEthSignedMessageHash(newValCheckpoint)
        currentValSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
    })

})