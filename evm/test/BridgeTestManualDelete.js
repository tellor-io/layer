const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const BN = ethers.BigNumber.from
const abiCoder = new ethers.utils.AbiCoder();

describe("BlobstreamO Manual - Function Tests", function () {

    let bridge, valPower, accounts, validators, powers, initialValAddrs, initialPowers, nonce, threshold, valCheckpoint;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks

    beforeEach(async function () {
        accounts = await ethers.getSigners();
        initialValAddrs = [accounts[1].address, accounts[2].address]
        initialPowers = [1, 2]
        nonce = 1
        threshold = 2
        valCheckpoint = h.calculateValCheckpoint(initialValAddrs, initialPowers, nonce, threshold)

        const Bridge = await ethers.getContractFactory("BlobstreamO");
        bridge = await Bridge.deploy(nonce, threshold, valCheckpoint);
        await bridge.deployed();
    });

    it("constructor", async function () {
        valSet = [accounts[0].address, accounts[1].address]
        powers = [1, 2]
        nonce = 1
        threshold = 1
        valCheckpoint = h.calculateValCheckpoint(valSet, powers, nonce, threshold)
        console.log("valCheckpoint: ", valCheckpoint)
    })

    it("updateValidatorSet", async function() {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, nonce, newThreshold)
        newDigest = await h.getEthSignedMessageHash(newValCheckpoint)
        valSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        await bridge.updateValidatorSet(newValHash, newThreshold, valSetArray, sigStructArray);
    })

    it("verifyOracleData", async function() {
        queryId = h.hash("myquery")
        value = abiCoder.encode(["uint256"], [2000])
        blocky = await h.getBlock()
        timestamp = blocky.timestamp - 2
        consensusThreshold = 3
        blockTimestamp = timestamp + 1
        valHash = await h.calculateValHash(initialValAddrs, initialPowers)

        dataDigest = await h.getDataDigest(
            queryId, 
            value, 
            timestamp, 
            consensusThreshold, 
            nonce, 
            threshold, 
            valHash, 
            blockTimestamp
        )
        
        valSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(dataDigest))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(dataDigest))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        oracleDataStruct = await h.getOracleDataStruct(
            queryId, 
            value, 
            timestamp, 
            consensusThreshold, 
            nonce, 
            threshold, 
            valHash, 
            blockTimestamp
        )

        await bridge.verifyOracleData(
            oracleDataStruct, 
            valSetArray, 
            sigStructArray
        )
    })
    
})