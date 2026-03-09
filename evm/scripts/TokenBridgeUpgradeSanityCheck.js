require("@nomiclabs/hardhat-ethers");
require("dotenv").config();

const axios = require("axios");
const hre = require("hardhat");
const { ethers } = hre;

// Read-only sanity checker for TokenBridge V1 -> V2 migration progress.
// This script does not send transactions.
//
// Example:
// npx hardhat run scripts/TokenBridgeUpgradeSanityCheck.js --network mainnet
//
// Required env vars:
// - NODE_URL_MAINNET (or NODE_URL_SEPOLIA_TESTNET if running on sepolia)
// - LAYER_REST_URL
// - TOKEN_ADDRESS
// - TOKEN_BRIDGE_V1_ADDRESS
// - TOKEN_BRIDGE_V2_ADDRESS
// - INITIAL_DATA_BRIDGE_ADDRESS (the original data bridge used at V2 deployment)
//
// Optional env vars:
// - SYNTHETIC_WITHDRAW_ID (default: 1000000000000)
// - EXPECTED_SYNTH_RECIPIENT
// - EXPECTED_SYNTH_AMOUNT_LOYA
// - EXPECTED_TOKEN_BRIDGE_V2_DATA_BRIDGE

const TOKEN_BRIDGE_V1_ABI = [
  "function initialized() view returns (bool)",
  "function depositId() view returns (uint256)",
  "function bridgeState() view returns (uint8)",
  "function dataBridge() view returns (address)",
  "function tokensToClaim(address) view returns (uint256)",
  "function withdrawClaimed(uint256) view returns (bool)",
];

const TOKEN_BRIDGE_V2_ABI = [
  "function initialized() view returns (bool)",
  "function depositId() view returns (uint256)",
  "function bridgeState() view returns (uint8)",
  "function bridgeStateUpdateTime() view returns (uint256)",
  "function lastPauseTimestamp() view returns (uint256)",
  "function dataBridge() view returns (address)",
  "function totalPauseTributeBalance() view returns (uint256)",
  "function withdrawClaimed(uint256) view returns (bool)",
  "function roles(bytes32) view returns (address roleAddress, uint256 roleUpdateDelay)",
];

const ERC20_ABI = ["function balanceOf(address) view returns (uint256)"];

function mustEnv(name) {
  const value = process.env[name];
  if (!value || value.trim() === "") {
    throw new Error(`Missing required env var: ${name}`);
  }
  return value.trim();
}

function optionalEnv(name) {
  const value = process.env[name];
  if (!value || value.trim() === "") {
    return null;
  }
  return value.trim();
}

function toWeiBigNumber(value, label) {
  try {
    return ethers.BigNumber.from(value);
  } catch (e) {
    throw new Error(`Invalid numeric value for ${label}: ${value}`);
  }
}

function toHexQueryId(queryType, isDepositReport, id) {
  const encodedArgs = ethers.utils.defaultAbiCoder.encode(["bool", "uint256"], [isDepositReport, id]);
  const encoded = ethers.utils.defaultAbiCoder.encode(["string", "bytes"], [queryType, encodedArgs]);
  return ethers.utils.keccak256(encoded);
}

function formatTrb(weiAmount) {
  return `${ethers.utils.formatEther(weiAmount)} TRB`;
}

function icon(done) {
  return done ? "✅" : "⏳";
}

function normalizeAddress(addr) {
  return ethers.utils.getAddress(addr);
}

function decodeAggregateValue(rawValue) {
  if (!rawValue || typeof rawValue !== "string") {
    return null;
  }

  let bytesLike = rawValue.trim();
  if (!bytesLike) {
    return null;
  }

  if (!bytesLike.startsWith("0x")) {
    // Layer endpoints often return hex-without-prefix for bytes fields.
    if (/^[0-9a-fA-F]+$/.test(bytesLike)) {
      bytesLike = `0x${bytesLike}`;
    } else {
      // Fallback for base64 payloads.
      try {
        bytesLike = `0x${Buffer.from(bytesLike, "base64").toString("hex")}`;
      } catch (e) {
        return null;
      }
    }
  }

  try {
    const decoded = ethers.utils.defaultAbiCoder.decode(
      ["address", "string", "uint256", "uint256"],
      bytesLike
    );
    return {
      recipient: decoded[0],
      layerSender: decoded[1],
      amountLoya: decoded[2],
      tipLoya: decoded[3],
      encodedValue: bytesLike,
    };
  } catch (e) {
    return null;
  }
}

async function queryLayerJson(layerRestUrl, path) {
  const url = `${layerRestUrl.replace(/\/$/, "")}${path}`;
  try {
    const res = await axios.get(url, { timeout: 12000 });
    return { ok: true, data: res.data, status: res.status, url };
  } catch (err) {
    return {
      ok: false,
      data: null,
      status: err.response?.status || null,
      error: err.message,
      url,
    };
  }
}

function pickNumericField(obj, keys) {
  for (const key of keys) {
    if (obj && obj[key] !== undefined && obj[key] !== null) {
      return String(obj[key]);
    }
  }
  return null;
}

async function fetchLayerState(layerRestUrl, syntheticWithdrawId, syntheticLegacyQueryIdNoPrefix) {
  const [lastWithdrawalRes, currentValTsRes] = await Promise.all([
    queryLayerJson(layerRestUrl, "/layer/bridge/get_last_withdrawal_id"),
    queryLayerJson(layerRestUrl, "/layer/bridge/get_current_validator_set_timestamp"),
  ]);

  const syntheticRes = await queryLayerJson(
    layerRestUrl,
    `/tellor-io/layer/oracle/get_current_aggregate_report/${syntheticLegacyQueryIdNoPrefix}`
  );

  const layerState = {
    lastWithdrawalId: null,
    currentValidatorTimestamp: null,
    syntheticAggregate: {
      found: false,
      timestamp: null,
      aggregateValueRaw: null,
      decodedValue: null,
      fetchError: null,
      fetchStatus: null,
    },
  };

  if (lastWithdrawalRes.ok) {
    layerState.lastWithdrawalId = pickNumericField(lastWithdrawalRes.data, [
      "withdrawal_id",
      "withdrawalId",
    ]);
  }

  if (currentValTsRes.ok) {
    layerState.currentValidatorTimestamp = pickNumericField(currentValTsRes.data, ["timestamp"]);
  }

  if (syntheticRes.ok) {
    const aggregateObj = syntheticRes.data?.aggregate || syntheticRes.data;
    const aggregateValue =
      aggregateObj?.aggregate_value ??
      aggregateObj?.aggregateValue ??
      syntheticRes.data?.aggregate_value ??
      null;
    const aggregateTimestamp = pickNumericField(syntheticRes.data, ["timestamp"]);

    layerState.syntheticAggregate.found = true;
    layerState.syntheticAggregate.timestamp = aggregateTimestamp;
    layerState.syntheticAggregate.aggregateValueRaw = aggregateValue;
    layerState.syntheticAggregate.decodedValue = decodeAggregateValue(aggregateValue);
  } else {
    layerState.syntheticAggregate.fetchError = syntheticRes.error;
    layerState.syntheticAggregate.fetchStatus = syntheticRes.status;
  }

  return layerState;
}

async function main() {
  const rpcUrl =
    // optionalEnv("NODE_URL_MAINNET") ||
    optionalEnv("NODE_URL_SEPOLIA_TESTNET") ||
    optionalEnv("NODE_URL");
  if (!rpcUrl) {
    throw new Error("Missing RPC URL: set NODE_URL_MAINNET or NODE_URL_SEPOLIA_TESTNET or NODE_URL");
  }

  const layerRestUrl = mustEnv("LAYER_REST_URL");
  const tokenAddress = normalizeAddress(mustEnv("TOKEN_ADDRESS"));
  const v1Address = normalizeAddress(mustEnv("TOKEN_BRIDGE_V1_ADDRESS"));
  const v2Address = normalizeAddress(mustEnv("TOKEN_BRIDGE_V2_ADDRESS"));
  const initialDataBridgeAddress = normalizeAddress(mustEnv("INITIAL_DATA_BRIDGE_ADDRESS"));
  const expectedV2DataBridgeAddress = optionalEnv("EXPECTED_TOKEN_BRIDGE_V2_DATA_BRIDGE");
  const expectedSyntheticRecipient = optionalEnv("EXPECTED_SYNTH_RECIPIENT");
  const expectedSyntheticAmountLoya = optionalEnv("EXPECTED_SYNTH_AMOUNT_LOYA");
  const syntheticWithdrawId = toWeiBigNumber(optionalEnv("SYNTHETIC_WITHDRAW_ID") || "1000000000000", "SYNTHETIC_WITHDRAW_ID");

  const provider = new ethers.providers.JsonRpcProvider(rpcUrl);
  const v1 = new ethers.Contract(v1Address, TOKEN_BRIDGE_V1_ABI, provider);
  const v2 = new ethers.Contract(v2Address, TOKEN_BRIDGE_V2_ABI, provider);
  const trb = new ethers.Contract(tokenAddress, ERC20_ABI, provider);

  const syntheticLegacyQueryId = toHexQueryId("TRBBridge", false, syntheticWithdrawId);
  const syntheticLegacyQueryIdNoPrefix = syntheticLegacyQueryId.slice(2);

  const layerState = await fetchLayerState(layerRestUrl, syntheticWithdrawId, syntheticLegacyQueryIdNoPrefix);

  const [
    latestBlock,
    v2Code,
    v1Initialized,
    v1DepositId,
    v1BridgeState,
    v1DataBridge,
    v1TokensToClaimForV2,
    v1SyntheticClaimed,
    v2Initialized,
    v2DepositId,
    v2BridgeState,
    v2BridgeStateUpdateTime,
    v2LastPauseTimestamp,
    v2DataBridge,
    v2PauseTributeBalance,
    v1TrbBalance,
    v2TrbBalance,
    mainGuardianRole,
    approvePauseRole,
    updateDataBridgeRole,
  ] = await Promise.all([
    provider.getBlock("latest"),
    provider.getCode(v2Address),
    v1.initialized(),
    v1.depositId(),
    v1.bridgeState(),
    v1.dataBridge(),
    v1.tokensToClaim(v2Address),
    v1.withdrawClaimed(syntheticWithdrawId),
    v2.initialized(),
    v2.depositId(),
    v2.bridgeState(),
    v2.bridgeStateUpdateTime(),
    v2.lastPauseTimestamp(),
    v2.dataBridge(),
    v2.totalPauseTributeBalance(),
    trb.balanceOf(v1Address),
    trb.balanceOf(v2Address),
    v2.roles(ethers.utils.keccak256(ethers.utils.toUtf8Bytes("MAIN_GUARDIAN"))),
    v2.roles(ethers.utils.keccak256(ethers.utils.toUtf8Bytes("APPROVE_PAUSE"))),
    v2.roles(ethers.utils.keccak256(ethers.utils.toUtf8Bytes("UPDATE_DATA_BRIDGE"))),
  ]);

  let v2CursorCoversLayerLastWithdrawal = null;
  if (layerState.lastWithdrawalId !== null) {
    v2CursorCoversLayerLastWithdrawal = await v2.withdrawClaimed(layerState.lastWithdrawalId);
  }

  const bridgeStateNameV2 =
    Number(v2BridgeState) === 0 ? "UNPAUSED" : Number(v2BridgeState) === 1 ? "PAUSED" : `UNKNOWN(${v2BridgeState})`;
  const bridgeStateNameV1 =
    Number(v1BridgeState) === 0
      ? "NORMAL"
      : Number(v1BridgeState) === 1
      ? "PAUSED"
      : Number(v1BridgeState) === 2
      ? "UNPAUSED"
      : `UNKNOWN(${v1BridgeState})`;

  const decodedSynthetic = layerState.syntheticAggregate.decodedValue;
  const syntheticRecipientMatches =
    expectedSyntheticRecipient && decodedSynthetic
      ? normalizeAddress(decodedSynthetic.recipient) === normalizeAddress(expectedSyntheticRecipient)
      : null;
  const syntheticAmountMatches =
    expectedSyntheticAmountLoya && decodedSynthetic
      ? decodedSynthetic.amountLoya.eq(toWeiBigNumber(expectedSyntheticAmountLoya, "EXPECTED_SYNTH_AMOUNT_LOYA"))
      : null;

  const checks = [
    {
      label: "TokenBridgeV2 deployed",
      done: v2Code && v2Code !== "0x",
      detail: `code size ${v2Code === "0x" ? 0 : (v2Code.length - 2) / 2} bytes`,
    },
    {
      label: "TokenBridgeV2 initialized",
      done: !!v2Initialized,
      detail: `v2 depositId=${v2DepositId.toString()}`,
    },
    {
      label: "V2 init cursor includes Layer last withdrawal id",
      done: v2CursorCoversLayerLastWithdrawal === true,
      detail:
        layerState.lastWithdrawalId === null
          ? "could not query Layer withdrawal cursor"
          : `layer lastWithdrawalId=${layerState.lastWithdrawalId}`,
    },
    {
      label: "Synthetic legacy withdraw report exists on Layer",
      done: layerState.syntheticAggregate.found,
      detail: `legacy queryId=${syntheticLegacyQueryId}`,
    },
    {
      label: "Synthetic legacy withdraw relayed on V1",
      done: !!v1SyntheticClaimed,
      detail: `withdrawClaimed(${syntheticWithdrawId.toString()})=${v1SyntheticClaimed}`,
    },
    {
      label: "Bridge pause step executed",
      done: Number(v2BridgeState) === 1,
      detail: `v2 bridgeState=${bridgeStateNameV2}`,
    },
    {
      label: "DataBridge rotated on V2",
      done:
        normalizeAddress(v2DataBridge) !== normalizeAddress(initialDataBridgeAddress) &&
        (!expectedV2DataBridgeAddress || normalizeAddress(v2DataBridge) === normalizeAddress(expectedV2DataBridgeAddress)),
      detail: `v2 dataBridge=${v2DataBridge}`,
    },
    {
      label: "Post-pause unpause executed",
      done: Number(v2BridgeState) === 0 && !v2LastPauseTimestamp.isZero(),
      detail: `v2 bridgeState=${bridgeStateNameV2}, lastPauseTimestamp=${v2LastPauseTimestamp.toString()}`,
    },
    {
      label: "V1 -> V2 draining path active",
      done: v1SyntheticClaimed || !v1TokensToClaimForV2.isZero(),
      detail: `v1.tokensToClaim[v2]=${formatTrb(v1TokensToClaimForV2)}`,
    },
  ];

  console.log("\nToken Bridge Upgrade Sanity Check (read-only)");
  console.log("================================================");
  console.log(`Network: ${hre.network.name}`);
  console.log(`EVM latest block: ${latestBlock.number} (${latestBlock.timestamp})`);
  console.log(`Layer REST: ${layerRestUrl}`);

  console.log("\nAddresses");
  console.log("---------");
  console.log(`Token:       ${tokenAddress}`);
  console.log(`Bridge V1:   ${v1Address}`);
  console.log(`Bridge V2:   ${v2Address}`);
  console.log(`Init DataBridge (expected old): ${initialDataBridgeAddress}`);
  if (expectedV2DataBridgeAddress) {
    console.log(`Expected new DataBridge:        ${normalizeAddress(expectedV2DataBridgeAddress)}`);
  }

  console.log("\nStep Status");
  console.log("-----------");
  for (const check of checks) {
    console.log(`${icon(check.done)} ${check.label} - ${check.detail}`);
  }

  console.log("\nEVM Snapshot");
  console.log("------------");
  console.log(`V1 initialized: ${v1Initialized}`);
  console.log(`V2 initialized: ${v2Initialized}`);
  console.log(`V1 bridgeState: ${bridgeStateNameV1}`);
  console.log(`V2 bridgeState: ${bridgeStateNameV2}`);
  console.log(`V1 depositId:   ${v1DepositId.toString()}`);
  console.log(`V2 depositId:   ${v2DepositId.toString()}`);
  console.log(`V1 dataBridge:  ${v1DataBridge}`);
  console.log(`V2 dataBridge:  ${v2DataBridge}`);
  console.log(`V1 TRB balance: ${formatTrb(v1TrbBalance)}`);
  console.log(`V2 TRB balance: ${formatTrb(v2TrbBalance)}`);
  console.log(`V2 pause tribute escrow: ${formatTrb(v2PauseTributeBalance)}`);
  console.log(`V2 bridgeStateUpdateTime: ${v2BridgeStateUpdateTime.toString()}`);
  console.log(`V2 lastPauseTimestamp:    ${v2LastPauseTimestamp.toString()}`);

  console.log("\nV2 Roles");
  console.log("--------");
  console.log(`MAIN_GUARDIAN:    ${mainGuardianRole.roleAddress} (delay=${mainGuardianRole.roleUpdateDelay.toString()})`);
  console.log(`APPROVE_PAUSE:    ${approvePauseRole.roleAddress} (delay=${approvePauseRole.roleUpdateDelay.toString()})`);
  console.log(`UPDATE_DATA_BRIDGE: ${updateDataBridgeRole.roleAddress} (delay=${updateDataBridgeRole.roleUpdateDelay.toString()})`);

  console.log("\nLayer Snapshot");
  console.log("--------------");
  console.log(
    `Layer last withdrawal id: ${
      layerState.lastWithdrawalId === null ? "unavailable" : layerState.lastWithdrawalId
    }`
  );
  console.log(
    `Layer current valset timestamp: ${
      layerState.currentValidatorTimestamp === null ? "unavailable" : layerState.currentValidatorTimestamp
    }`
  );
  if (!layerState.syntheticAggregate.found) {
    console.log(
      `Synthetic aggregate lookup failed (status=${layerState.syntheticAggregate.fetchStatus}): ${layerState.syntheticAggregate.fetchError}`
    );
  } else {
    console.log(
      `Synthetic aggregate timestamp: ${
        layerState.syntheticAggregate.timestamp === null ? "unavailable" : layerState.syntheticAggregate.timestamp
      }`
    );
    console.log(`Synthetic aggregate raw value: ${layerState.syntheticAggregate.aggregateValueRaw}`);
    if (decodedSynthetic) {
      console.log("Synthetic aggregate decoded:");
      console.log(`  recipient: ${decodedSynthetic.recipient}`);
      console.log(`  layerSender: ${decodedSynthetic.layerSender}`);
      console.log(`  amount (loya): ${decodedSynthetic.amountLoya.toString()}`);
      console.log(`  tip (loya): ${decodedSynthetic.tipLoya.toString()}`);
      if (syntheticRecipientMatches !== null) {
        console.log(`  expected recipient match: ${syntheticRecipientMatches}`);
      }
      if (syntheticAmountMatches !== null) {
        console.log(`  expected amount match: ${syntheticAmountMatches}`);
      }
    } else {
      console.log("Synthetic aggregate value could not be ABI-decoded as (address,string,uint256,uint256).");
    }
  }

  const doneCount = checks.filter((c) => c.done).length;
  console.log("\nProgress");
  console.log("--------");
  console.log(`${doneCount}/${checks.length} sanity milestones detected as complete.`);
  console.log("Manual steps (reporter rollout, relayer ops, adversarial test execution) still need operator confirmation.");
}

main()
  .then(() => process.exit(0))
  .catch((err) => {
    console.error("\nSanity check failed:");
    console.error(err);
    process.exit(1);
  });
