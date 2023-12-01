const { expect } = require("chai");
const { ethers, network } = require("hardhat");
const h = require("./helpers/helpers");
var assert = require('assert');
const web3 = require('web3');
const { prependOnceListener } = require("process");
const BN = ethers.BigNumber.from
const abiCoder = new ethers.utils.AbiCoder();

describe("Blobstream - Function Tests", function () {

    let bridge, validatorHash, valPower, accounts, validators, powers;
    let startHeight = 0;
    const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks
    const CHAIN_ID = "layer"
    

    beforeEach(async function () {
        const Bridge = await ethers.getContractFactory("Blobstream");
        if(startHeight == 0) {
            startHeight = await h.getLatestBlockNumber()
            valsResponse = await h.getValidatorSet(startHeight)
            validators = valsResponse[0]
            powers = valsResponse[1]
        }
        for(i = 0; i< powers.length; i++){
            valPower += powers[i]
        }
        let enc = ethers.utils.defaultAbiCoder.encode(["address[]"], validators)
        let validatorHash = web3.utils.keccak256(enc);
        bridge = await Bridge.deploy(startHeight,valPower * 2/3,validatorHash) ;
        accounts = await ethers.getSigners();
    });

    it("Should be able to set the bridge address", async function () {
        assert.equal(1, 1)
        height = await h.getLatestBlockNumber()
        console.log("height: " + height)
        let [vals, powers] = await h.getValidatorSet(height)
        console.log("vals: " + vals)
        console.log("powers: " + powers)
        // await bridge.setBridgeAddress(accounts[0].address);
        // expect(await bridge.bridgeAddress()).to.equal(accounts[0].address);
    })

    it("test verifyBlockHeader", async function() {
        multistoreData = await h.getMultistore()
        console.log("muultistore: ", multistoreData)
        merkleParts = await h.getBlockHeaderMerkleParts(startHeight)
        console.log("merkleParts: ", merkleParts)
        commonParts = await h.getCommonEncodedVoteParts(startHeight)
        console.log("commonParts: ", commonParts)
        tmSig = await h.getTmSig(startHeight)
        console.log("tmSig: ", tmSig)
        // result = await bridge.verifyBlockHeader(multistoreData, merkleParts, commonParts, tmSig)
        // result = await bridge.readMultistoreData(multistoreData)
        a = BigInt("100")
        b = BigInt("200")
        // encode as struct 
        // struct FakeStruct {
        //     uint a;
        //     uint b;
        // }
        // fakeEnc = abiCoder.encode(["tuple(uint256, uint256)"], [[a, b]])
        // fakeEnc = abiCoder.encode(["uint256", "uint256"], [a, b])
        // result = await bridge.readFakeStruct(fakeEnc)

        // try web3 for encoding fake struct 
        // web3.eth.abi.encodeParameter(
        //     {
        //         "ParentStruct": {
        //             "propertyOne": 'uint256',
        //             "propertyTwo": 'uint256',
        //             "childStruct": {
        //                 "propertyOne": 'uint256',
        //                 "propertyTwo": 'uint256'
        //             }
        //         }
        //     },
        //     {
        //         "propertyOne": 42,
        //         "propertyTwo": 56,
        //         "childStruct": {
        //             "propertyOne": 45,
        //             "propertyTwo": 78
        //         }
        //     }
        // );
        fakeEnc = web3.eth.abi.encodeParameter(
            {
                "FakeStruct": {
                    "a": 'uint256',
                    "b": 'uint256'
                }
            },
            {
                "a": a,
                "b": b
            }
        );
        // result = await bridge.readFakeStruct(fakeEnc)
        // dec = ethers.utils.defaultAbiCoder.decode(["uint256", "uint256"], fakeEnc)

        types = ["uint256", "uint256"]
        vals = [a, b]
        fakeEnc = ethers.utils.defaultAbiCoder.encode(types, vals)
        // fakeEnc = ethers.utils.defaultAbiCoder.encode(["uint256", "uint256"], [[a, b]])
        // result = await bridge.readFakeStructBytes(fakeEnc)

        // bridge.FakeStruct()

        aEnc = abiCoder.encode(["uint256"], [a])
        bEnc = abiCoder.encode(["uint256"], [b])
        fakeEnc = {
            a: aEnc,
            b: bEnc
        }

        result = await bridge.readFakeStruct(fakeEnc)

        multistoreData = await h.getMultistore()
        console.log("multistoreData: ", multistoreData)
        result = await bridge.readMultistoreData(multistoreData)
        console.log("result: ", result)

    })

    it.only("test verifyBlockHeader", async function() {
        multistoreData = await h.getMultistore(startHeight)
        merklePartsData = await h.getBlockHeaderMerkleParts(startHeight)
        commonPartsData = await h.getCommonEncodedVoteParts(startHeight)
        tmSigData = await h.getTmSig(startHeight)
        result = await bridge.verifyBlockHeader(multistoreData, merklePartsData, commonPartsData, tmSigData)

        console.log("multistore: ", multistoreData)
        console.log("merkleParts: ", merklePartsData)
        console.log("commonParts: ", commonPartsData)
        console.log("tmSig: ", tmSigData)

        assert.equal(result, true)
    })
    
})