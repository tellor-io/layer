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

    it("init test", async function () {
        valSet = [accounts[0].address, accounts[1].address]
        powers = [1, 2]
        nonce = 1
        threshold = 1
        valCheckpoint = h.calculateValCheckpoint(valSet, powers, nonce, threshold)
        console.log("valCheckpoint: ", valCheckpoint)
    })

    it.only("updateValidatorSet", async function() {
        newValAddrs = [accounts[1].address, accounts[2].address, accounts[3].address]
        newPowers = [1, 2, 3]
        newNonce = 2
        newThreshold = 4
        newValHash = await h.calculateValHash(newValAddrs, newPowers)
        newValCheckpoint = h.calculateValCheckpoint(newValAddrs, newPowers, nonce, newThreshold)
        newDigest = await h.getEthSignedMessageHash(newValCheckpoint)
        valSetArray = await h.getValSetStructArray(initialValAddrs, initialPowers)
        sig1 = await accounts[1].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sig2 = await accounts[2].signMessage(ethers.utils.arrayify(newValCheckpoint))
        sigStructArray = await h.getSigStructArray([sig1, sig2])
        console.log("sig1: ", sig1)
        console.log("sig2: ", sig2)
        // resp = await bridge.deleteThisInputValSet([{addr: accounts[1].address, power: 1}, {addr: accounts[2].address, power: 2}])
        tryRecoverResp = await bridge.tryRecoverPublic(newDigest, sig1)
        console.log("tryRecoverResp: ", tryRecoverResp)
        console.log("addr1: ", accounts[1].address)
        console.log("testNewCheckpoint: ", newValCheckpoint)
        await bridge.updateValidatorSet(newValHash, newThreshold, valSetArray, sigStructArray);
    })

    it("sigs", async function() {
        const signer = accounts[1]
        message0 = 'Hello, world!'
        messageHash0 = ethers.utils.hashMessage(message0)
        mysig = await signer.signMessage(message0)
        console.log("mysig: ", mysig)
        tryRecoverResp = await bridge.tryRecoverPublic(messageHash0, mysig)
        console.log("tryRecoverResp: ", tryRecoverResp)
        console.log("addr1: ", accounts[1].address)

        console.log("\n\nBREAK\n\n")
        const message = 'Hello, world!';
        const messageHash = ethers.utils.hashMessage(message);
        console.log("messageHash: ", messageHash)
        console.log("message type: ", typeof(message))
        console.log("messageHash type: ", typeof(messageHash))
        const signature = await signer.signMessage(message);
        console.log("sig type: ", typeof(signature))
        console.log("signature: ", signature)

        // Pass the hashed message and signature to tryRecover
        const recoveredAddress = await bridge.tryRecoverPublic(messageHash, signature);

        console.log('Original address:', signer.address);
        console.log('Recovered address:', recoveredAddress);

        
    })

    

   
    
})