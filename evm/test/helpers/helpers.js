const web3 = require('web3');
const { ethers } = require("hardhat");
const BigNumber = ethers.BigNumber
const BN = web3.utils.BN;
const axios = require('axios')

const hash = web3.utils.keccak256;
var assert = require('assert');
const abiCoder = new ethers.utils.AbiCoder();

getLatestBlockNumber = async () => {
    url = "http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest"
    try {
        const response = await axios.get(url)
        latest_block_number = response.data.block.header.height
        return latest_block_number
    } catch (error) {
        console.log(error)
    }
}


getValidatorSet = async (height) => {
    url = "http://localhost:1317/layer/bridge/blockvalidators?height=" + height
    try {
        const response = await axios.get(url)
        validators = []
        powers = []
        for (i = 0; i < response.data.Validators.length; i++) {
            validators.push(response.data.Validators[i].ethAddress)
            powers.push(response.data.Validators[i].votingPower)
        }
        console.log("success")
        return [validators, powers]
    } catch (error) {
        console.log("fail")
        console.log(error)
    }
}

getMultistore = async (height) => {
    url = "http://localhost:1317/layer/bridge/multistore?height=" + height
    try {
        const response = await axios.get(url)
        multistoreResp = response.data.MutiStoreTree
        enc = {
          oracleIAVLStateHash: abiCoder.encode(["bytes32"], ["0x" + multistoreResp.oracle_iavl_state_hash]),
          paramsStoreMerkleHash: abiCoder.encode(["bytes32"], ["0x" + multistoreResp.mint_store_merkle_hash]),
          slashingToStakingStoresMerkleHash: abiCoder.encode(["bytes32"], ["0x" + multistoreResp.icacontroller_to_icahost_merkle_hash]),
          govToMintStoresMerkleHash: abiCoder.encode(["bytes32"], ["0x" + multistoreResp.feegrant_to_ibc_merkle_hash]),
          authToFeegrantStoresMerkleHash: abiCoder.encode(["bytes32"], ["0x" + multistoreResp.acc_to_evidence_merkle_hash]),
          transferToUpgradeStoresMerkleHash: abiCoder.encode(["bytes32"], ["0x" + multistoreResp.params_to_vesting_merkle_hash])
        }
        return enc
    } catch (error) {
        console.log(error)
    }
}

// struct BlockDetail {
//   bytes32 oracleHash;
//   uint64 timeSecond;
//   uint32 timeNanoSecondFraction;
// }

// enum ResolveStatus {
//   RESOLVE_STATUS_OPEN_UNSPECIFIED,
//   RESOLVE_STATUS_SUCCESS,
//   RESOLVE_STATUS_FAILURE,
//   RESOLVE_STATUS_EXPIRED
// }

// struct Value {
//   string clientID;
//   uint64 oracleScriptID;
//   bytes params;
//   uint64 askCount;
//   uint64 minCount;
//   uint64 requestID;
//   uint64 ansCount;
//   uint64 requestTime;
//   uint64 resolveTime;
//   ResolveStatus resolveStatus;
//   bytes result;
// }

// struct MultistoreData {
//   bytes32 oracleIAVLStateHash; // [C]
//   bytes32 paramsStoreMerkleHash; // [D]
//   bytes32 slashingToStakingStoresMerkleHash; // [I7]
//   bytes32 govToMintStoresMerkleHash; // [I10]
//   bytes32 authToFeegrantStoresMerkleHash; // [I12]
//   bytes32 transferToUpgradeStoresMerkleHash; // [I15]
// }

// struct BlockHeaderMerkleParts {
//   bytes32 versionAndChainIdHash; // [1A]
//   uint64 height; // [2]
//   uint64 timeSecond; // [3]
//   uint32 timeNanoSecondFraction; // between 0 to 10^9 [3]
//   bytes32 lastBlockIdAndOther; // [2B]
//   bytes32 nextValidatorHashAndConsensusHash; // [1E]
//   bytes32 lastResultsHash; // [B]
//   bytes32 evidenceAndProposerHash; // [2D]
// }

// struct CommonEncodedVotePartData {
//   bytes signedDataPrefix;
//   bytes signedDataSuffix;
// }

// struct TMSignatureData {
//   bytes32 r;
//   bytes32 s;
//   uint8 v;
//   bytes encodedTimestamp;
// }

// struct IAVLMerklePath {
//   bool isDataOnRight;
//   uint8 subtreeHeight;
//   uint256 subtreeSize;
//   uint256 subtreeVersion;
//   bytes32 siblingHash;
// }

getBlockHeaderMerkleParts = async (height) => {
  url = "http://localhost:1317/layer/bridge/blockheadermerkleevm?height=" + height
  try {
      const response = await axios.get(url)
      headerParts = response.data.blockheaderMerkleEvm
      enc = {
        versionAndChainIdHash: abiCoder.encode(["bytes32"], [headerParts.versionChainidHash]),
        height: abiCoder.encode(["uint64"], [headerParts.height]),
        timeSecond: abiCoder.encode(["uint64"], [headerParts.timeSecond]),
        timeNanoSecondFraction: abiCoder.encode(["uint32"], [headerParts.timeNanosecond]),
        lastBlockIdAndOther: abiCoder.encode(["bytes32"], [headerParts.lastblockidCommitHash]),
        nextValidatorHashAndConsensusHash: abiCoder.encode(["bytes32"], [headerParts.nextvalidatorConsensusHash]),
        lastResultsHash: abiCoder.encode(["bytes32"], [headerParts.lastresultsHash]),
        evidenceAndProposerHash: abiCoder.encode(["bytes32"], [headerParts.evidenceProposerHash])
      }
      return enc
  } catch (error) {
      console.log(error)
  }
} 

getCommonEncodedVoteParts = async (height) => {
  url = "http://localhost:1317/layer/bridge/tmsig?height=" + height
  try {
      const response = await axios.get(url)
      common = response.data.common
      // convert Base64 common parts to hex
      prefix = "0x" + Buffer.from(common.SignedDataPrefix, 'base64').toString('hex')
      suffix = "0x" + Buffer.from(common.SignedDataSuffix, 'base64').toString('hex')
      notenc = {
        signedDataPrefix: prefix,
        signedDataSuffix: suffix
      }
      return notenc
  } catch (error) {
      console.log(error)
  }
}

getTmSig = async (height) => {
  url = "http://localhost:1317/layer/bridge/tmsig?height=" + height
  try {
      const response = await axios.get(url)
      tmSig = response.data.tmSig
      tmSigEnc = []
      for (i = 0; i < tmSig.length; i++) {
          tmSigEnc.push({
            r: abiCoder.encode(["bytes32"], [tmSig[i].R]),
            s: abiCoder.encode(["bytes32"], [tmSig[i].S]),
            v: abiCoder.encode(["uint8"], [tmSig[i].V]),
            // encodedTimestamp: abiCoder.encode(["bytes"], [tmSig[i].EncodedTimestamp])
            encodedTimestamp: removeLeadingZeros(tmSig[i].EncodedTimestamp)
          })
      }
      return tmSigEnc
  } catch (error) {
      console.log(error)
  }
}

function removeLeadingZeros(hexString) {
  let result = hexString.replace(/^0x0*/, '0x');
  if ((result.length - 2) % 2 !== 0) {
      result = result.replace('0x', '0x0');
  }
  return result;
}

advanceTimeAndBlock = async (time) => {
    await advanceTime(time);
    await advanceBlock();
    console.log("Time Travelling...");
    return Promise.resolve(web3.eth.getBlock("latest"));
  };

const takeFifteen = async () => {
await advanceTime(60 * 18);
};

advanceTime = async (time) =>{
  await network.provider.send("evm_increaseTime", [time])
  await network.provider.send("evm_mine")
}

advanceBlock = () => {
  return new Promise((resolve, reject) => {
    web3.currentProvider.send(
      {
        jsonrpc: "2.0",
        method: "evm_mine",
        id: new Date().getTime(),
      },
      (err, result) => {
        if (err) {
          return reject(err);
        }
        const newBlockHash = web3.eth.getBlock("latest").hash;

        return resolve(newBlockHash);
      }
    );
  });
};

async function expectThrow(promise) {
  try {
    await promise;
  } catch (error) {
    const invalidOpcode = error.message.search("invalid opcode") >= 0;
    const outOfGas = error.message.search("out of gas") >= 0;
    const revert = error.message.search("revert") >= 0;
    assert(
      invalidOpcode || outOfGas || revert,
      "Expected throw, got '" + error + "' instead"
    );
    return;
  }
  assert.fail("Expected throw not received");
}

function to18(n) {
  return ethers.BigNumber.from(n).mul(ethers.BigNumber.from(10).pow(18))
}

function tob32(n){
  return ethers.utils.formatBytes32String(n)
}

function uintTob32(n){
  let vars = web3.utils.toHex(n)
  vars = vars.slice(2)
  while(vars.length < 64){
    vars = "0" + vars
  }
  vars = "0x" + vars
  return vars
}

function bytes(n){
  return web3.utils.toHex(n)
}

function getBlock(){
  return ethers.provider.getBlock()
}

function toWei(n){
  return web3.utils.toWei(n, "ether")
}

function fromWei(n){
  return web3.utils.fromWei(n)
}

module.exports = {
  timeTarget: 240,
  hash,
  zeroAddress:"0x0000000000000000000000000000000000000000",
  to18,
  uintTob32,
  tob32,
  bytes,
  getBlock,
  advanceTime,
  advanceBlock,
  advanceTimeAndBlock,
  takeFifteen,
  toWei,
  fromWei,
  expectThrow,
  getLatestBlockNumber,
  getValidatorSet,
  getMultistore,
  getBlockHeaderMerkleParts,
  getCommonEncodedVoteParts,
  getTmSig,
};