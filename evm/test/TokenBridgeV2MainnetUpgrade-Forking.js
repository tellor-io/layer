const { expect } = require("chai");
const h = require("./helpers/evmHelpers");
var assert = require("assert");
const { ethers } = require("hardhat");

// Forking tests for upgrading the TokenBridgeV1 to the new TokenBridgeV2 contract on mainnet

describe("TokenBridge V1 -> V2 Upgrade - Forking Tests", function () {
  // Ethereum mainnet deployed contracts
  const TOKENBRIDGE_V1 = "0x5589e306b1920F009979a50B88caE32aecD471E4";
  const DATABRIDGE = "0xFfa3393BE1E4b442fff6cD0df0794B0031e9CF65";

  // Mainnet Tellor contracts (used by the bridges)
  const TELLOR_MASTER = "0x88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0";
  const TELLORFLEX = "0x8cFc184c877154a8F9ffE0fe75649dbe5e2DBEbf";

  // Layer constants (matching other fork tests)
  const BIG_WITHDRAW_ID = ethers.BigNumber.from("1000000000000"); // matches upgrade-plan notes
  const UNBONDING_PERIOD = 86400 * 7 * 3; // 3 weeks
  const TOKEN_DECIMAL_PRECISION_MULTIPLIER = ethers.BigNumber.from("1000000000000"); // 1e12
  const VALIDATOR_SET_DOMAIN_SEPARATOR_MAINNET =
    "0x636865636b706f696e7400000000000000000000000000000000000000000000";
  const abiCoder = new ethers.utils.AbiCoder();

  let accounts = null;
  let snapshot = null;
  let tbridgeV1 = null;
  let tbridgeV2 = null;
  let dataBridge = null;
  let tellor = null;

  before(async function () {
    snapshot = await h.takeSnapshot();
    this.timeout(300000);
  });

  after(async function () {
    // Leave fork in a clean state for other suites.
    await snapshot.restore();
  });

  beforeEach(async function () {
    await snapshot.restore();
    accounts = await ethers.getSigners();

    tbridgeV1 = await ethers.getContractAt("TokenBridge", TOKENBRIDGE_V1);
    dataBridge = await ethers.getContractAt("TellorDataBridge", DATABRIDGE);
    tellor = await ethers.getContractAt(
      "contracts/tellor360/oldContracts/contracts/interfaces/ITellor.sol:ITellor",
      TELLOR_MASTER
    );

    // Deploy V2 pointing at mainnet Tellor + mainnet DataBridge
    tbridgeV2 = await ethers.deployContract("TokenBridgeV2", [TELLOR_MASTER, DATABRIDGE, TELLORFLEX]);
  });

  it("upgrade: deploy V2, init ids, drain V1 -> V2 via synthetic withdraw + claimExtraWithdraw", async function () {
    this.timeout(300000);

    // --- sanity checks on mainnet V1 ---
    assert.equal(await tbridgeV1.dataBridge(), DATABRIDGE, "V1 should point to the mainnet DataBridge");
    assert.equal(await tbridgeV1.token(), TELLOR_MASTER, "V1 should use Tellor token");
    assert.equal(await tbridgeV2.initialized(), false, "V2 should start uninitialized");
    await h.expectThrow(tbridgeV2.depositToLayer(h.toWei("1"), h.toWei("0"), "layer"));// not initialized

    // --- init V2 to continue ids from V1 (upgrade step) ---
    const v1DepositId = await tbridgeV1.depositId();
    await tbridgeV2.init(v1DepositId, v1DepositId);
    assert.equal(await tbridgeV2.initialized(), true, "V2 should be initialized after init()");
    assert.equal((await tbridgeV2.depositId()).toString(), v1DepositId.toString(), "V2 depositId should match V1 at upgrade");
    if (v1DepositId.gt(0)) {
      assert.equal(await tbridgeV2.withdrawClaimed(v1DepositId.sub(1)), true, "V2 should mark prior withdraw ids claimed");
    }

    // --- reset mainnet DataBridge validator set so we can sign attestations in this fork ---
    const guardianAddr = await dataBridge.guardian();
    await accounts[0].sendTransaction({ to: guardianAddr, value: ethers.utils.parseEther("1") });
    await h.impersonateAccount(guardianAddr);
    const guardian = await ethers.provider.getSigner(guardianAddr);

    const unbondingBn = ethers.BigNumber.from(UNBONDING_PERIOD);
    await h.advanceTime(unbondingBn.add(2).toNumber());

    // choose a new validator set and ensure timestamp increases
    const val1 = ethers.Wallet.createRandom();
    const val2 = ethers.Wallet.createRandom();
    const valAddrs = [val1.address, val2.address];
    const powers = [1, 2];
    const threshold = 2;

    const block1 = await h.getBlock();
    let newValTimestamp = ethers.BigNumber.from(block1.timestamp).sub(2).mul(1000); // ms
   
    const newValHash = await h.calculateValHash(valAddrs, powers);
    const valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, newValTimestamp, VALIDATOR_SET_DOMAIN_SEPARATOR_MAINNET);
    await dataBridge.connect(guardian).guardianResetValidatorSet(threshold, newValTimestamp, valCheckpoint);

    // --- synthetic withdraw on V1 sending funds to V2 ---
    // Use legacy query type TRBBridge (V1) and a very large withdraw id (matches upgrade-plan notes).
    const WITHDRAW_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, BIG_WITHDRAW_ID]);
    const WITHDRAW_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW_QUERY_DATA_ARGS]);
    const WITHDRAW_QUERY_ID = h.hash(WITHDRAW_QUERY_DATA);

    const v1BalBefore = await tellor.balanceOf(TOKENBRIDGE_V1);
    assert(v1BalBefore.gt(h.toWei("10000")), "V1 should have at least 10,000 TRB");

    // request a withdraw larger than the 5% limit so remainder becomes tokensToClaim[V2]
    const requestedWei = v1BalBefore.div(2);
    const amountLoya = requestedWei.div(TOKEN_DECIMAL_PRECISION_MULTIPLIER); // 1e12 multiplier
    assert(amountLoya.gt(0), "amountLoya should be > 0");
    const amountConverted = amountLoya.mul(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    const withdrawValue = h.getWithdrawValue(tbridgeV2.address, "layer", amountLoya);

    const block2 = await h.getBlock();
    const reportTimestamp = ethers.BigNumber.from(block2.timestamp).sub(43201).mul(1000); // >12h old, in ms
    const attestTimestamp = ethers.BigNumber.from(block2.timestamp).mul(1000); // now, in ms
    const aggregatePower = 3;
    const previousTimestamp = 0;
    const nextTimestamp = 0;
    const lastConsensusTimestamp = reportTimestamp;

    const dataDigest = await h.getDataDigest(
      WITHDRAW_QUERY_ID,
      withdrawValue,
      reportTimestamp,
      aggregatePower,
      previousTimestamp,
      nextTimestamp,
      valCheckpoint,
      attestTimestamp,
      lastConsensusTimestamp
    );
    const sig1 = await h.layerSign(dataDigest, val1.privateKey);
    const sig2 = await h.layerSign(dataDigest, val2.privateKey);
    const currentValSetArray = await h.getValSetStructArray(valAddrs, powers);
    const sigStructArray = await h.getSigStructArray([sig1, sig2]);
    const oracleDataStruct = await h.getOracleDataStruct(
      WITHDRAW_QUERY_ID,
      withdrawValue,
      reportTimestamp,
      aggregatePower,
      previousTimestamp,
      nextTimestamp,
      attestTimestamp,
      lastConsensusTimestamp
    );

    const v2Bal0 = await tellor.balanceOf(tbridgeV2.address);
    await tbridgeV1.withdrawFromLayer(oracleDataStruct, currentValSetArray, sigStructArray, BIG_WITHDRAW_ID);
    const v2Bal1 = await tellor.balanceOf(tbridgeV2.address);
    const v1BalAfter = await tellor.balanceOf(TOKENBRIDGE_V1);

    // In V1, withdraw limit is 5% per 12h. Since we advanced 3 weeks, the limit refreshes and uses current balance.
    const expectedLimit1 = v1BalBefore.div(20);
    const expectedPaid1 = expectedLimit1.lt(amountConverted) ? expectedLimit1 : amountConverted;
    assert.equal(v2Bal1.sub(v2Bal0).toString(), expectedPaid1.toString(), "V2 tranche1 should match withdraw limit math");
    assert.equal(v1BalBefore.sub(v1BalAfter).toString(), expectedPaid1.toString(), "V1 balance decrease should equal tranche1");
    assert.equal(await tbridgeV1.withdrawClaimed(BIG_WITHDRAW_ID), true, "V1 withdraw id should be marked claimed");

    const toClaim = await tbridgeV1.tokensToClaim(tbridgeV2.address);
    const expectedToClaim = amountConverted.sub(expectedPaid1);
    assert.equal(toClaim.toString(), expectedToClaim.toString(), "V1 tokensToClaim[V2] should equal remainder");
    assert(toClaim.gt(0), "V1 should allocate remainder to tokensToClaim[V2]");

    // --- claim subsequent tranches via claimExtraWithdraw every 12 hours ---
    await h.advanceTime(43200 + 2);
    const v2Bal2a = await tellor.balanceOf(tbridgeV2.address);
    const v1Bal2a = await tellor.balanceOf(TOKENBRIDGE_V1);
    await tbridgeV1.claimExtraWithdraw(tbridgeV2.address);
    const v2Bal2b = await tellor.balanceOf(tbridgeV2.address);
    const v1Bal2b = await tellor.balanceOf(TOKENBRIDGE_V1);
    const expectedLimit2 = v1Bal2a.div(20);
    const expectedPaid2 = expectedLimit2.lt(toClaim) ? expectedLimit2 : toClaim;
    assert.equal(v2Bal2b.sub(v2Bal2a).toString(), expectedPaid2.toString(), "V2 tranche2 should match withdraw limit math");
    assert.equal(v1Bal2a.sub(v1Bal2b).toString(), expectedPaid2.toString(), "V1 balance decrease should equal tranche2");

    const toClaimAfter1 = await tbridgeV1.tokensToClaim(tbridgeV2.address);
    assert.equal(toClaimAfter1.toString(), toClaim.sub(expectedPaid2).toString(), "tokensToClaim should decrease by tranche2");
    assert(toClaimAfter1.lt(toClaim), "tokensToClaim should decrease after claimExtraWithdraw");
  });

  it("upgrade: after draining, can withdraw from TokenBridgeV2 (TRBBridgeV2)", async function () {
    this.timeout(300000);

    // init V2 to continue ids from V1 (upgrade step)
    const v1DepositId = await tbridgeV1.depositId();
    await tbridgeV2.init(v1DepositId, v1DepositId);

    // reset DataBridge validator set so we can sign attestations in this fork
    const guardianAddr = await dataBridge.guardian();
    await accounts[0].sendTransaction({ to: guardianAddr, value: ethers.utils.parseEther("1") });
    await h.impersonateAccount(guardianAddr);
    const guardian = await ethers.provider.getSigner(guardianAddr);
    await h.advanceTime(ethers.BigNumber.from(UNBONDING_PERIOD).add(2).toNumber());

    const val1 = ethers.Wallet.createRandom();
    const val2 = ethers.Wallet.createRandom();
    const valAddrs = [val1.address, val2.address];
    const powers = [1, 2];
    const threshold = 2;
    const block1 = await h.getBlock();
    const newValTimestamp = ethers.BigNumber.from(block1.timestamp).sub(2).mul(1000); // ms
    const newValHash = await h.calculateValHash(valAddrs, powers);
    const valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, newValTimestamp, VALIDATOR_SET_DOMAIN_SEPARATOR_MAINNET);
    await dataBridge.connect(guardian).guardianResetValidatorSet(threshold, newValTimestamp, valCheckpoint);

    // seed V2 with a tranche from V1 (synthetic legacy withdraw to V2)
    const WITHDRAW_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, BIG_WITHDRAW_ID]);
    const WITHDRAW_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW_QUERY_DATA_ARGS]);
    const WITHDRAW_QUERY_ID = h.hash(WITHDRAW_QUERY_DATA);

    const v1BalBefore = await tellor.balanceOf(TOKENBRIDGE_V1);
    assert(v1BalBefore.gt(h.toWei("10000")), "V1 should have at least 10,000 TRB");
    const amountLoya = v1BalBefore.div(2).div(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    const withdrawValue = h.getWithdrawValue(tbridgeV2.address, "layer", amountLoya);

    const block2 = await h.getBlock();
    const reportTimestamp = ethers.BigNumber.from(block2.timestamp).sub(43201).mul(1000);
    const attestTimestamp = ethers.BigNumber.from(block2.timestamp).mul(1000);
    const aggregatePower = 3;
    const dataDigest = await h.getDataDigest(
      WITHDRAW_QUERY_ID,
      withdrawValue,
      reportTimestamp,
      aggregatePower,
      0,
      0,
      valCheckpoint,
      attestTimestamp,
      reportTimestamp
    );
    const sig1 = await h.layerSign(dataDigest, val1.privateKey);
    const sig2 = await h.layerSign(dataDigest, val2.privateKey);
    const currentValSetArray = await h.getValSetStructArray(valAddrs, powers);
    const sigStructArray = await h.getSigStructArray([sig1, sig2]);
    const oracleDataStruct = await h.getOracleDataStruct(
      WITHDRAW_QUERY_ID,
      withdrawValue,
      reportTimestamp,
      aggregatePower,
      0,
      0,
      attestTimestamp,
      reportTimestamp
    );
    await tbridgeV1.withdrawFromLayer(oracleDataStruct, currentValSetArray, sigStructArray, BIG_WITHDRAW_ID);

    const v2BalBefore = await tellor.balanceOf(tbridgeV2.address);
    assert(v2BalBefore.gt(0), "V2 should have tokens after seeding from V1");

    // now withdraw from V2 to a recipient using TRBBridgeV2 query type
    const recipient = accounts[3].address;
    const withdrawIdV2 = BIG_WITHDRAW_ID.add(123);
    const amountWei = v2BalBefore.div(25);
    const amountLoya2 = amountWei.div(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    const amountConverted2 = amountLoya2.mul(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    assert(amountConverted2.gt(0), "amountConverted2 should be > 0");

    const WITHDRAW2_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, withdrawIdV2]);
    const WITHDRAW2_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridgeV2", WITHDRAW2_QUERY_DATA_ARGS]);
    const WITHDRAW2_QUERY_ID = h.hash(WITHDRAW2_QUERY_DATA);
    const value2 = h.getWithdrawValue(recipient, "layer", amountLoya2);

    const block3 = await h.getBlock();
    const reportTimestamp2 = ethers.BigNumber.from(block3.timestamp).sub(43201).mul(1000);
    const attestTimestamp2 = ethers.BigNumber.from(block3.timestamp).mul(1000);
    const dataDigest2 = await h.getDataDigest(
      WITHDRAW2_QUERY_ID,
      value2,
      reportTimestamp2,
      aggregatePower,
      0,
      0,
      valCheckpoint,
      attestTimestamp2,
      reportTimestamp2
    );
    const sig1b = await h.layerSign(dataDigest2, val1.privateKey);
    const sig2b = await h.layerSign(dataDigest2, val2.privateKey);
    const sigStructArray2 = await h.getSigStructArray([sig1b, sig2b]);
    const oracleDataStruct2 = await h.getOracleDataStruct(
      WITHDRAW2_QUERY_ID,
      value2,
      reportTimestamp2,
      aggregatePower,
      0,
      0,
      attestTimestamp2,
      reportTimestamp2
    );

    const userBal0 = await tellor.balanceOf(recipient);
    await tbridgeV2.withdrawFromLayer(oracleDataStruct2, currentValSetArray, sigStructArray2, withdrawIdV2);
    const userBal1 = await tellor.balanceOf(recipient);
    const v2BalAfter = await tellor.balanceOf(tbridgeV2.address);
    assert.equal(userBal1.sub(userBal0).toString(), amountConverted2.toString(), "recipient should receive full withdraw amount");
    assert.equal(v2BalBefore.sub(v2BalAfter).toString(), amountConverted2.toString(), "V2 balance should decrease by withdraw amount");
    assert.equal(await tbridgeV2.withdrawClaimed(withdrawIdV2), true, "V2 withdraw id should be marked claimed");
  });

  it("upgrade: V2 withdraw rate-limit + claimExtraWithdraw works for user remainder", async function () {
    this.timeout(300000);

    // init V2 and reset DataBridge validator set
    const v1DepositId = await tbridgeV1.depositId();
    await tbridgeV2.init(v1DepositId, v1DepositId);
    const guardianAddr = await dataBridge.guardian();
    await accounts[0].sendTransaction({ to: guardianAddr, value: ethers.utils.parseEther("1") });
    await h.impersonateAccount(guardianAddr);
    const guardian = await ethers.provider.getSigner(guardianAddr);
    await h.advanceTime(ethers.BigNumber.from(UNBONDING_PERIOD).add(2).toNumber());

    const val1 = ethers.Wallet.createRandom();
    const val2 = ethers.Wallet.createRandom();
    const valAddrs = [val1.address, val2.address];
    const powers = [1, 2];
    const threshold = 2;
    const block1 = await h.getBlock();
    const newValTimestamp = ethers.BigNumber.from(block1.timestamp).sub(2).mul(1000); // ms
    const newValHash = await h.calculateValHash(valAddrs, powers);
    const valCheckpoint = h.calculateValCheckpoint(newValHash, threshold, newValTimestamp, VALIDATOR_SET_DOMAIN_SEPARATOR_MAINNET);
    await dataBridge.connect(guardian).guardianResetValidatorSet(threshold, newValTimestamp, valCheckpoint);

    // seed V2 with a tranche from V1
    const WITHDRAW_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, BIG_WITHDRAW_ID]);
    const WITHDRAW_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridge", WITHDRAW_QUERY_DATA_ARGS]);
    const WITHDRAW_QUERY_ID = h.hash(WITHDRAW_QUERY_DATA);
    const amountLoya = (await tellor.balanceOf(TOKENBRIDGE_V1)).div(2).div(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    const withdrawValue = h.getWithdrawValue(tbridgeV2.address, "layer", amountLoya);
    const block2 = await h.getBlock();
    const reportTimestamp = ethers.BigNumber.from(block2.timestamp).sub(43201).mul(1000);
    const attestTimestamp = ethers.BigNumber.from(block2.timestamp).mul(1000);
    const aggregatePower = 3;
    const dataDigest = await h.getDataDigest(
      WITHDRAW_QUERY_ID,
      withdrawValue,
      reportTimestamp,
      aggregatePower,
      0,
      0,
      valCheckpoint,
      attestTimestamp,
      reportTimestamp
    );
    const sig1 = await h.layerSign(dataDigest, val1.privateKey);
    const sig2 = await h.layerSign(dataDigest, val2.privateKey);
    const currentValSetArray = await h.getValSetStructArray(valAddrs, powers);
    const sigStructArray = await h.getSigStructArray([sig1, sig2]);
    const oracleDataStruct = await h.getOracleDataStruct(
      WITHDRAW_QUERY_ID,
      withdrawValue,
      reportTimestamp,
      aggregatePower,
      0,
      0,
      attestTimestamp,
      reportTimestamp
    );
    await tbridgeV1.withdrawFromLayer(oracleDataStruct, currentValSetArray, sigStructArray, BIG_WITHDRAW_ID);

    // withdraw from V2 with an amount > 5% limit, creating tokensToClaim for the recipient
    const recipient = accounts[4].address;
    const withdrawIdV2 = BIG_WITHDRAW_ID.add(456);
    const v2BalBefore = await tellor.balanceOf(tbridgeV2.address);
    const amountWei = v2BalBefore.div(2);
    const amountLoya2 = amountWei.div(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    const amountConverted2 = amountLoya2.mul(TOKEN_DECIMAL_PRECISION_MULTIPLIER);
    assert(amountConverted2.gt(0), "amountConverted2 should be > 0");

    const WITHDRAW2_QUERY_DATA_ARGS = abiCoder.encode(["bool", "uint256"], [false, withdrawIdV2]);
    const WITHDRAW2_QUERY_DATA = abiCoder.encode(["string", "bytes"], ["TRBBridgeV2", WITHDRAW2_QUERY_DATA_ARGS]);
    const WITHDRAW2_QUERY_ID = h.hash(WITHDRAW2_QUERY_DATA);
    const value2 = h.getWithdrawValue(recipient, "layer", amountLoya2);

    const block3 = await h.getBlock();
    const reportTimestamp2 = ethers.BigNumber.from(block3.timestamp).sub(43201).mul(1000);
    const attestTimestamp2 = ethers.BigNumber.from(block3.timestamp).mul(1000);
    const dataDigest2 = await h.getDataDigest(
      WITHDRAW2_QUERY_ID,
      value2,
      reportTimestamp2,
      aggregatePower,
      0,
      0,
      valCheckpoint,
      attestTimestamp2,
      reportTimestamp2
    );
    const sig1b = await h.layerSign(dataDigest2, val1.privateKey);
    const sig2b = await h.layerSign(dataDigest2, val2.privateKey);
    const sigStructArray2 = await h.getSigStructArray([sig1b, sig2b]);
    const oracleDataStruct2 = await h.getOracleDataStruct(
      WITHDRAW2_QUERY_ID,
      value2,
      reportTimestamp2,
      aggregatePower,
      0,
      0,
      attestTimestamp2,
      reportTimestamp2
    );

    const expectedLimit1 = v2BalBefore.div(20);
    const expectedPaid1 = expectedLimit1.lt(amountConverted2) ? expectedLimit1 : amountConverted2;

    const userBal0 = await tellor.balanceOf(recipient);
    await tbridgeV2.withdrawFromLayer(oracleDataStruct2, currentValSetArray, sigStructArray2, withdrawIdV2);
    const userBal1 = await tellor.balanceOf(recipient);
    const v2BalAfter = await tellor.balanceOf(tbridgeV2.address);
    assert.equal(userBal1.sub(userBal0).toString(), expectedPaid1.toString(), "recipient should receive rate-limited tranche1");
    assert.equal(v2BalBefore.sub(v2BalAfter).toString(), expectedPaid1.toString(), "V2 balance should decrease by tranche1");

    const toClaim0 = await tbridgeV2.tokensToClaim(recipient);
    assert.equal(toClaim0.toString(), amountConverted2.sub(expectedPaid1).toString(), "V2 tokensToClaim should equal remainder");

    // claim next tranche after 12h
    await h.advanceTime(43200 + 2);
    const v2BalClaim0 = await tellor.balanceOf(tbridgeV2.address);
    const userBalClaim0 = await tellor.balanceOf(recipient);
    await tbridgeV2.claimExtraWithdraw(recipient);
    const v2BalClaim1 = await tellor.balanceOf(tbridgeV2.address);
    const userBalClaim1 = await tellor.balanceOf(recipient);

    const expectedLimit2 = v2BalClaim0.div(20);
    const expectedPaid2 = expectedLimit2.lt(toClaim0) ? expectedLimit2 : toClaim0;
    assert.equal(userBalClaim1.sub(userBalClaim0).toString(), expectedPaid2.toString(), "recipient should receive tranche2 via claimExtraWithdraw");
    assert.equal(v2BalClaim0.sub(v2BalClaim1).toString(), expectedPaid2.toString(), "V2 balance should decrease by tranche2");
    const toClaim1 = await tbridgeV2.tokensToClaim(recipient);
    assert.equal(toClaim1.toString(), toClaim0.sub(expectedPaid2).toString(), "V2 tokensToClaim should decrease by tranche2");
  });
});

