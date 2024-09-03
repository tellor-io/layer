const web3 = require('web3');
const { ethers, network } = require("hardhat");
const BigNumber = ethers.BigNumber
const BN = web3.utils.BN;
const axios = require('axios')
const { DirectSecp256k1HdWallet, Registry } = require('@cosmjs/proto-signing');
const { GasPrice, StargateClient, SigningStargateClient } = require('@cosmjs/stargate');
const os = require('os');
const path = require('path');
const fs = require('fs').promises;
const { MsgRequestAttestations } = require('../../generated/layer/bridge/tx_pb.js');
const { impersonateAccount, takeSnapshot } = require("@nomicfoundation/hardhat-network-helpers");

const homeDirectory = os.homedir();
const CHARLIE_MNEMONIC_FILE = path.join(homeDirectory, 'Desktop', 'charlie_mnemonic.txt');


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

getValsetTimestampByIndex = async (index) => {
  url = "http://localhost:1317/layer/bridge/get_validator_timestamp_by_index/" + index
  try {
    const response = await axios.get(url)
    timestamp = response.data.timestamp
    return timestamp
  } catch (error) {
    console.log(error)
  }
}

getValsetCheckpointParams = async (timestamp) => {
  url = "http://localhost:1317/layer/bridge/get_validator_checkpoint_params/" + timestamp
  try {
    const response = await axios.get(url)
    checkpointParams = {
      checkpoint: '0x' + response.data.checkpoint,
      valsetHash: '0x' + response.data.valset_hash,
      timestamp: response.data.timestamp,
      powerThreshold: response.data.power_threshold
    }
    return checkpointParams
  } catch (error) {
    console.log(error)
  }
}

getValset = async (timestamp) => {
  url = "http://localhost:1317/layer/bridge/get_valset_by_timestamp/" + timestamp
  try {
    const response = await axios.get(url)
    valsetResponse = response.data.bridge_validator_set
    valset = []
    for (i = 0; i < valsetResponse.length; i++) {
      valset.push({
        addr: valsetResponse[i].ethereumAddress,
        power: valsetResponse[i].power
      })
    }
    return valset
  } catch (error) {
    console.log(error)
  }
}

getValsetSigs = async (timestamp, valset, checkpoint) => {
  const url = "http://localhost:1317/layer/bridge/get_valset_sigs/" + timestamp;
  try {
    const response = await axios.get(url);
    const sigsResponse = response.data.signatures;
    const sigs = [];
    // get sha256 hash of the message
    const messageHash = ethers.utils.sha256(checkpoint);
    for (let i = 0; i < sigsResponse.length; i++) {
      const signature = sigsResponse[i];
      if (signature.length === 128) {
        // try v = 27
        let v = 27;
        let r = '0x' + signature.slice(0, 64);
        let s = '0x' + signature.slice(64, 128);
        let recoveredAddress = ethers.utils.recoverAddress(messageHash, {
          r: r,
          s: s,
          v: v,
        });
        // check if recovered address matches the validator address
        if (recoveredAddress.toLowerCase() !== valset[i].addr.toLowerCase()) {
          // try v = 28 if v = 27 did not match
          v = 28;
          recoveredAddress = ethers.utils.recoverAddress(messageHash, {
            r: r,
            s: s,
            v: v,
          });
          if (recoveredAddress.toLowerCase() !== valset[i].addr.toLowerCase()) {
            // If neither worked, use default values
            v = 0;
            r = '0x0000000000000000000000000000000000000000000000000000000000000000';
            s = '0x0000000000000000000000000000000000000000000000000000000000000000';
          }
        }
        sigs.push({
          v: v,
          r: '0x' + signature.slice(0, 64),
          s: '0x' + signature.slice(64, 128),
        });
      } else {
        sigs.push({
          v: 0,
          r: '0x0000000000000000000000000000000000000000000000000000000000000000',
          s: '0x0000000000000000000000000000000000000000000000000000000000000000',
        });
      }
    }
    return sigs;
  } catch (error) {
    console.log(error);
  }
}

getCurrentAggregateReport = async (queryId) => {
  const formattedQueryId = queryId.startsWith("0x") ? queryId.slice(2) : queryId;
  url = "http://localhost:1317/tellor-io/layer/oracle/get_current_aggregate_report/" + formattedQueryId
  try {
    const response = await axios.get(url)
    agg = response.data.aggregate
    ts = response.data.timestamp
    reportData = {
      value: "0x" + agg.aggregateValue,
      timestamp: ts,
      aggregatePower: agg.reporterPower,
      previousTimestamp: 0,
      nextTimestamp: 0
    }
    oracleAttestationData = {
      queryId: '0x' + formattedQueryId,
      report: reportData,
      attestTimestamp: ts
    }
    return oracleAttestationData
  } catch (error) {
    console.log(error)
  }
}

getDataBefore = async (queryId, timestamp) => {
  const formattedQueryId = queryId.startsWith("0x") ? queryId.slice(2) : queryId;
  url = "http://localhost:1317/tellor-io/layer/oracle/get_data_before/" + formattedQueryId + "/" + timestamp
  // http://tellornode.com:1317/tellor-io/layer/oracle/get_data_before/
  try {
    const response = await axios.get(url)
    agg = response.data.aggregate
    ts = response.data.timestamp
    reportData = {
      value: "0x" + agg.aggregateValue,
      timestamp: ts,
      aggregatePower: agg.reporterPower,
      previousTimestamp: 0,
      nextTimestamp: 0
    }
    return reportData
  } catch (error) {
    console.log(error)
  }
}

getOracleAttestations = async (queryId, timestamp, valset, digest) => {
  const formattedQueryId = queryId.startsWith("0x") ? queryId.slice(2) : queryId;
  const messageHash = ethers.utils.sha256(digest);
  url = "http://localhost:1317/layer/bridge/get_oracle_attestations/" + formattedQueryId + "/" + timestamp
  try {
    const response = await axios.get(url)
    attestsResponse = response.data.attestations
    attestations = []
    for (i = 0; i < attestsResponse.length; i++) {
      attestation = attestsResponse[i]
      if (attestation.length == 130) {
        // try v = 27
        let v = 27
        let r = '0x' + attestation.slice(2, 66)
        let s = '0x' + attestation.slice(66, 130)
        let recoveredAddress = ethers.utils.recoverAddress(messageHash, {
          r: r,
          s: s,
          v: v
        })
        if (recoveredAddress.toLowerCase() != valset[i].addr.toLowerCase()) {
          v = 28
          recoveredAddress = ethers.utils.recoverAddress(messageHash, {
            r: r,
            s: s,
            v: v
          })
          if (recoveredAddress.toLowerCase() != valset[i].addr.toLowerCase()) {
            v = 0;
            r = '0x0000000000000000000000000000000000000000000000000000000000000000';
            s = '0x0000000000000000000000000000000000000000000000000000000000000000';
          }
        }
        attestations.push({
          v: v,
          r: r,
          s: s
        })
      } else {
        attestations.push({
          v: 0,
          r: '0x0000000000000000000000000000000000000000000000000000000000000000',
          s: '0x0000000000000000000000000000000000000000000000000000000000000000'
        })
      }
    }
    return attestations
  } catch (error) {
    console.log(error)
  }
}

getSnapshotsByReport = async (queryId, timestamp) => {
  const formattedQueryId = queryId.startsWith("0x") ? queryId.slice(2) : queryId;
  url = "http://localhost:1317/layer/bridge/get_snapshots_by_report/" + formattedQueryId + "/" + timestamp
  try {
    const response = await axios.get(url)
    snapshots = response.data.snapshots
    return snapshots
  } catch (error) {
    // console.log(error)
  }
}

getAttestationDataBySnapshot = async (snapshot) => {
  url = "http://localhost:1317/layer/bridge/get_attestation_data_by_snapshot/" + snapshot
  try {
    const response = await axios.get(url)
    attestationDataReturned = response.data
    attestationData = {
      queryId: '0x' + attestationDataReturned.query_id,
      report: {
        value: '0x' + attestationDataReturned.aggregate_value,
        timestamp: attestationDataReturned.timestamp,
        aggregatePower: attestationDataReturned.aggregate_power,
        previousTimestamp: attestationDataReturned.previous_report_timestamp,
        nextTimestamp: attestationDataReturned.next_report_timestamp
      },
      attestationTimestamp: attestationDataReturned.attestation_timestamp
    }
    return attestationData
  } catch (error) {
    console.log(error)
  }
}

getAttestationsBySnapshot = async (snapshot, valset) => {
  url = "http://localhost:1317/layer/bridge/get_attestations_by_snapshot/" + snapshot
  const messageHash = ethers.utils.sha256('0x' + snapshot);
  try {
    const response = await axios.get(url)
    attestsResponse = response.data.attestations
    attestations = []
    for (i = 0; i < attestsResponse.length; i++) {
      attestation = '0x' + attestsResponse[i]
      if (attestation.length == 130) {
        // try v = 27
        let v = 27
        let r = '0x' + attestation.slice(2, 66)
        let s = '0x' + attestation.slice(66, 130)
        let recoveredAddress = ethers.utils.recoverAddress(messageHash, {
          r: r,
          s: s,
          v: v
        })
        if (recoveredAddress.toLowerCase() != valset[i].addr.toLowerCase()) {
          v = 28
          recoveredAddress = ethers.utils.recoverAddress(messageHash, {
            r: r,
            s: s,
            v: v
          })
          if (recoveredAddress.toLowerCase() != valset[i].addr.toLowerCase()) {
            v = 0;
            r = '0x0000000000000000000000000000000000000000000000000000000000000000';
            s = '0x0000000000000000000000000000000000000000000000000000000000000000';
          }
        }
        attestations.push({
          v: v,
          r: r,
          s: s
        })
      }
    }
    return attestations
  } catch (error) {
    console.log(error)
  }
}

createCosmosWallet = async () => {
  try {
    const mnemonic = await fs.readFile(CHARLIE_MNEMONIC_FILE, { encoding: 'utf8' });
    const trimmedMnemonic = mnemonic.trim();

    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(trimmedMnemonic, {
      prefix: 'tellor',
    });

    const [firstAccount] = await wallet.getAccounts();

    console.log(`Wallet address: ${firstAccount.address}`);
    return wallet;
  } catch (error) {
    console.error('Failed to create wallet from mnemonic:', error);
  }
}

requestAttestations = async (queryId, timestamp) => {
  const formattedQueryId = queryId.startsWith("0x") ? queryId.slice(2) : queryId;
  const rpcEndpoint = 'http://localhost:26657'; 
  const chainId = 'layer'; 

  const registry = new Registry();
  const typeUrl = "/layer.bridge.MsgRequestAttestations"
  registry.register(typeUrl, MsgRequestAttestations);
  const options = { registry: registry };

  // create a wallet with charlie's mnemonic
  const wallet = await createCosmosWallet();
  const [firstAccount] = await wallet.getAccounts();

  const client = await SigningStargateClient.connectWithSigner(rpcEndpoint, wallet, options);

  let msg = new MsgRequestAttestations()
  msg.setQueryid(formattedQueryId)
  msg.setTimestamp(timestamp)
  msg.setCreator(firstAccount.address)

  const fee = {
    amount: [{ denom: 'stake', amount: '2000' }],
    gas: '200000',
  };

  // sign and broadcast the transaction
  console.log("signing and broadcasting")
  const result = await client.signAndBroadcast(firstAccount.address, [msg], fee, 'Request attestations');
  assertIsBroadcastTxSuccess(result);

  console.log('Transaction result:', result);
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

calculateValCheckpoint = (valSet, powers, threshold, valTimestamp) => {
  structArray = []
  for (i = 0; i < valSet.length; i++) {
    structArray[i] = {
      addr: valSet[i],
      power: powers[i]
    }
  }
  // encode the array of Validator struct objects into bytes so they can be hashed
  enc = abiCoder.encode(["tuple(address addr, uint256 power)[]"], [structArray])
  // hash the encoded bytes
  valHash = hash(enc)

  domainSeparator = "0x636865636b706f696e7400000000000000000000000000000000000000000000"
  enc = abiCoder.encode(["bytes32", "uint256", "uint256", "bytes32"], [domainSeparator, threshold, valTimestamp, valHash])
  valCheckpoint = hash(enc)
  return valCheckpoint
}

calculateValHash = (valSet, powers) => {
  structArray = []
  for (i = 0; i < valSet.length; i++) {
    structArray[i] = {
      addr: valSet[i],
      power: powers[i]
    }
  }
  // encode the array of Validator struct objects into bytes so they can be hashed
  enc = abiCoder.encode(["tuple(address addr, uint256 power)[]"], [structArray])
  // hash the encoded bytes
  valHash = hash(enc)
  return valHash
}

getEthSignedMessageHash = (messageHash) => {
  const prefix = "\x19Ethereum Signed Message:\n32";
  const messageHashBytes = ethers.utils.arrayify(messageHash);
  const prefixBytes = ethers.utils.toUtf8Bytes(prefix);
  const combined = ethers.utils.concat([prefixBytes, messageHashBytes]);
  const digest = ethers.utils.keccak256(combined);
  return digest;
}

getDataDigest = (queryId, value, timestamp, aggregatePower, previousTimestamp, nextTimestamp, valCheckpoint, blockTimestamp) => {
  const DOMAIN_SEPARATOR = "0x74656c6c6f7243757272656e744174746573746174696f6e0000000000000000"
  enc = abiCoder.encode(["bytes32", "bytes32", "bytes", "uint256", "uint256", "uint256", "uint256", "bytes32", "uint256"],
    [DOMAIN_SEPARATOR, queryId, value, timestamp, aggregatePower, previousTimestamp, nextTimestamp, valCheckpoint, blockTimestamp])
  return hash(enc)
}

domainSeparateOracleAttestationData = (attestationData, valCheckpoint) => {
  const DOMAIN_SEPARATOR = "0x74656c6c6f7243757272656e744174746573746174696f6e0000000000000000"
  enc = abiCoder.encode(["bytes32", "bytes32", "bytes", "uint256", "uint256", "uint256", "uint256", "bytes32", "uint256"],
    [DOMAIN_SEPARATOR, attestationData.queryId, attestationData.report.value, attestationData.report.timestamp, attestationData.report.aggregatePower, attestationData.report.previousTimestamp, attestationData.report.nextTimestamp, valCheckpoint, attestationData.attestTimestamp])

  // print everything
  console.log("DOMAIN_SEPARATOR", DOMAIN_SEPARATOR)
  console.log("attestationData.queryId", attestationData.queryId)
  console.log("attestationData.report.value", attestationData.report.value)
  console.log("attestationData.report.timestamp", attestationData.report.timestamp)
  console.log("attestationData.report.aggregatePower", attestationData.report.aggregatePower)
  console.log("attestationData.report.previousTimestamp", attestationData.report.previousTimestamp)
  console.log("attestationData.report.nextTimestamp", attestationData.report.nextTimestamp)
  console.log("valCheckpoint", valCheckpoint)
  console.log("attestationData.attestTimestamp", attestationData.attestTimestamp)
  console.log("dataDigest", hash(enc))
  console.log("enc", enc)
  return hash(enc)
}

getValSetStructArray = (valAddrs, powers) => {
  structArray = []
  for (i = 0; i < valAddrs.length; i++) {
    structArray[i] = {
      addr: valAddrs[i],
      power: powers[i]
    }
  }
  return structArray
}

getSigStructArray = (sigs) => {
  structArray = []
  for (i = 0; i < sigs.length; i++) {
    let { v, r, s } = ethers.utils.splitSignature(sigs[i])
    structArray[i] = {
      v: abiCoder.encode(["uint8"], [v]),
      r: abiCoder.encode(["bytes32"], [r]),
      s: abiCoder.encode(["bytes32"], [s])
    }
  }
  return structArray
}

getOracleDataStruct = (queryId, value, timestamp, aggregatePower, previousTimestamp, nextTimestamp, attestTimestamp) => {
  return {
    queryId: queryId,
    report: {
      value: value,
      timestamp: timestamp,
      aggregatePower: aggregatePower,
      previousTimestamp: previousTimestamp,
      nextTimestamp: nextTimestamp
    },
    attestTimestamp: attestTimestamp
  }
}

layerSign = (message, privateKey) => {
  // assumes message is bytesLike
  messageHash = ethers.utils.sha256(message)
  signingKey = new ethers.utils.SigningKey(privateKey)
  signature = signingKey.signDigest(messageHash)
  return signature
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

advanceTime = async (time) => {
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

function tob32(n) {
  return ethers.utils.formatBytes32String(n)
}

function uintTob32(n) {
  let vars = web3.utils.toHex(n)
  vars = vars.slice(2)
  while (vars.length < 64) {
    vars = "0" + vars
  }
  vars = "0x" + vars
  return vars
}

function bytes(n) {
  return web3.utils.toHex(n)
}

function getBlock() {
  return ethers.provider.getBlock()
}

function toWei(n) {
  return web3.utils.toWei(n, "ether")
}

function fromWei(n) {
  return web3.utils.fromWei(n)
}

function sleep(s) {
  return new Promise(resolve => setTimeout(resolve, s * 1000));
}

module.exports = {
  timeTarget: 240,
  hash,
  zeroAddress: "0x0000000000000000000000000000000000000000",
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
  sleep,
  getLatestBlockNumber,
  getValidatorSet,
  calculateValCheckpoint,
  calculateValHash,
  getEthSignedMessageHash,
  getValSetStructArray,
  getSigStructArray,
  getOracleDataStruct,
  getDataDigest,
  getValsetTimestampByIndex,
  getValsetCheckpointParams,
  getValset,
  getValsetSigs,
  getCurrentAggregateReport,
  getDataBefore,
  getOracleAttestations,
  domainSeparateOracleAttestationData,
  getSnapshotsByReport,
  getAttestationDataBySnapshot,
  getAttestationsBySnapshot,
  requestAttestations,
  createCosmosWallet,
  impersonateAccount,
  takeSnapshot,
  layerSign
};

