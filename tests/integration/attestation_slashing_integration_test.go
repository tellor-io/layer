package integration_test

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	bridgekeeper "github.com/tellor-io/layer/x/bridge/keeper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) TestAttestationSlashingIntegration() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(10).WithBlockTime(time.Now())

	// create validators with real staking
	numValidators := 3
	validatorPowers := []uint64{1000, 2000, 3000} // different powers for testing

	valAccAddrs, valAddrs, privKeys := s.createValidatorsWithEVMKeys(numValidators, validatorPowers)

	// setup bridge validator checkpoints
	startTime := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("test-checkpoint-123"),
		ValsetHash:     []byte("valset-hash-456"),
		Timestamp:      startTime,
		PowerThreshold: uint64(2000), // 2/3 of total power (6000)
		BlockHeight:    10,
	}
	require.NoError(s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Set(ctx, startTime, checkpointParams))

	// create bridge validator set for historical power lookup
	bridgeValset := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: make([]*bridgetypes.BridgeValidator, numValidators),
	}

	for i := 0; i < numValidators; i++ {
		// get EVM address for validator
		evmAddr, err := s.Setup.Bridgekeeper.OperatorToEVMAddressMap.Get(ctx, valAddrs[i].String())
		require.NoError(err)

		bridgeValset.BridgeValidatorSet[i] = &bridgetypes.BridgeValidator{
			EthereumAddress: evmAddr.EVMAddress,
			Power:           validatorPowers[i],
		}
	}

	require.NoError(s.Setup.Bridgekeeper.BridgeValsetByTimestampMap.Set(ctx, startTime, bridgeValset))

	// record initial validator states
	initialStates := make([]ValidatorState, numValidators)
	for i, valAddr := range valAddrs {
		validator, err := s.Setup.Stakingkeeper.GetValidator(ctx, valAddr)
		require.NoError(err)

		initialStates[i] = ValidatorState{
			Address: valAddr,
			Tokens:  validator.Tokens,
			Jailed:  validator.Jailed,
			Status:  validator.Status,
		}
	}

	// Test case 1: Submit valid attestation evidence and verify slashing
	targetValidatorIndex := 1 // slash the middle validator
	targetValidator := valAddrs[targetValidatorIndex]
	targetPrivKey := privKeys[targetValidatorIndex]

	// create malicious attestation data
	queryId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	maliciousValue := "000000000000000000000000000000000000000000000058528649cf90ee0000"
	attestTimestamp := uint64(time.Now().UnixMilli())

	// create and sign malicious attestation
	snapshotBytes, signature := s.createMaliciousAttestation(
		queryId,
		maliciousValue,
		attestTimestamp,
		checkpointParams.Checkpoint,
		targetPrivKey,
	)

	// verify snapshot doesn't exist (making it malicious)
	exists, err := s.Setup.Bridgekeeper.SnapshotToAttestationsMap.Has(ctx, snapshotBytes)
	require.NoError(err)
	require.False(exists, "Snapshot should not exist to be considered malicious")

	// submit attestation evidence
	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg := &bridgetypes.MsgSubmitAttestationEvidence{
		Creator:                valAccAddrs[0].String(), // submitter
		QueryId:                queryId,
		Value:                  maliciousValue,
		Timestamp:              attestTimestamp,
		AggregatePower:         uint64(100),
		PreviousTimestamp:      attestTimestamp - 1000,
		NextTimestamp:          attestTimestamp + 1000,
		ValsetCheckpoint:       hex.EncodeToString(checkpointParams.Checkpoint),
		AttestationTimestamp:   attestTimestamp,
		LastConsensusTimestamp: attestTimestamp,
		Signature:              signature,
	}

	_, err = bridgeMsgServer.SubmitAttestationEvidence(ctx, evidenceMsg)
	require.NoError(err, "Attestation evidence submission should succeed")

	// verify the validator was slashed
	slashedValidator, err := s.Setup.Stakingkeeper.GetValidator(ctx, targetValidator)
	require.NoError(err)

	// check that tokens were reduced (slashed)
	require.True(slashedValidator.Tokens.LT(initialStates[targetValidatorIndex].Tokens),
		"Validator should have been slashed (tokens reduced)")

	// check that validator was jailed
	require.True(slashedValidator.Jailed, "Validator should have been jailed")

	// verify other validators were not affected
	for i, valAddr := range valAddrs {
		if i == targetValidatorIndex {
			continue // skip the slashed validator
		}

		validator, err := s.Setup.Stakingkeeper.GetValidator(ctx, valAddr)
		require.NoError(err)
		require.Equal(initialStates[i].Tokens, validator.Tokens,
			"Other validators should not be slashed")
		require.False(validator.Jailed, "Other validators should not be jailed")
	}

	// verify evidence was recorded
	evidenceKey := collections.Join(targetValidator.Bytes(), attestTimestamp)
	evidenceExists, err := s.Setup.Bridgekeeper.AttestationEvidenceSubmitted.Has(ctx, evidenceKey)
	require.NoError(err)
	require.True(evidenceExists, "Evidence should be recorded")

	evidenceValue, err := s.Setup.Bridgekeeper.AttestationEvidenceSubmitted.Get(ctx, evidenceKey)
	require.NoError(err)
	require.True(evidenceValue.Submitted, "Evidence should be marked as submitted")
}

func (s *IntegrationTestSuite) TestAttestationSlashingRateLimit() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(20).WithBlockTime(time.Now())

	// create multiple validators for rate limit testing (since first one gets jailed)
	valAccAddrs, valAddrs, privKeys := s.createValidatorsWithEVMKeys(2, []uint64{1000, 1000})
	targetValidator1 := valAddrs[0]
	targetValidator2 := valAddrs[1]
	targetPrivKey1 := privKeys[0]
	targetPrivKey2 := privKeys[1]

	// setup checkpoint
	startTime := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("rate-limit-checkpoint"),
		ValsetHash:     []byte("rate-limit-hash"),
		Timestamp:      startTime,
		PowerThreshold: uint64(500),
		BlockHeight:    20,
	}
	require.NoError(s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Set(ctx, startTime, checkpointParams))

	// create bridge validator set for both validators
	evmAddr1, err := s.Setup.Bridgekeeper.OperatorToEVMAddressMap.Get(ctx, targetValidator1.String())
	require.NoError(err)
	evmAddr2, err := s.Setup.Bridgekeeper.OperatorToEVMAddressMap.Get(ctx, targetValidator2.String())
	require.NoError(err)

	bridgeValset := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
			{
				EthereumAddress: evmAddr1.EVMAddress,
				Power:           1000,
			},
			{
				EthereumAddress: evmAddr2.EVMAddress,
				Power:           1000,
			},
		},
	}
	require.NoError(s.Setup.Bridgekeeper.BridgeValsetByTimestampMap.Set(ctx, startTime, bridgeValset))

	// submit first attestation evidence (using first validator)
	attestTimestamp1 := uint64(time.Now().UnixMilli())
	queryId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"
	maliciousValue1 := "0000000000000000000000000000000000000000000000000000000000000001"

	snapshotBytes1, signature1 := s.createMaliciousAttestation(
		queryId,
		maliciousValue1,
		attestTimestamp1,
		checkpointParams.Checkpoint,
		targetPrivKey1,
	)

	// verify first snapshot doesn't exist
	exists, err := s.Setup.Bridgekeeper.SnapshotToAttestationsMap.Has(ctx, snapshotBytes1)
	require.NoError(err)
	require.False(exists)

	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg1 := &bridgetypes.MsgSubmitAttestationEvidence{
		Creator:                valAccAddrs[0].String(),
		QueryId:                queryId,
		Value:                  maliciousValue1,
		Timestamp:              attestTimestamp1,
		AggregatePower:         uint64(100),
		PreviousTimestamp:      attestTimestamp1 - 1000,
		NextTimestamp:          attestTimestamp1 + 1000,
		ValsetCheckpoint:       hex.EncodeToString(checkpointParams.Checkpoint),
		AttestationTimestamp:   attestTimestamp1,
		LastConsensusTimestamp: attestTimestamp1,
		Signature:              signature1,
	}

	_, err = bridgeMsgServer.SubmitAttestationEvidence(ctx, evidenceMsg1)
	require.NoError(err, "First attestation evidence should succeed")

	// try to submit second evidence within rate limit window (should fail)
	rateLimitMs, err := s.Setup.Bridgekeeper.GetAttestRateLimitWindow(ctx)
	require.NoError(err)
	require.Equal(uint64(10*60*1000), rateLimitMs) // 10 minutes default

	attestTimestamp2 := attestTimestamp1 + (rateLimitMs / 2) // 5 minutes later
	maliciousValue2 := "0000000000000000000000000000000000000000000000000000000000000002"

	_, signature2 := s.createMaliciousAttestation(
		queryId,
		maliciousValue2,
		attestTimestamp2,
		checkpointParams.Checkpoint,
		targetPrivKey1, // same validator for rate limit test
	)

	evidenceMsg2 := &bridgetypes.MsgSubmitAttestationEvidence{
		Creator:                valAccAddrs[0].String(),
		QueryId:                queryId,
		Value:                  maliciousValue2,
		Timestamp:              attestTimestamp2,
		AggregatePower:         uint64(100),
		PreviousTimestamp:      attestTimestamp2 - 1000,
		NextTimestamp:          attestTimestamp2 + 1000,
		ValsetCheckpoint:       hex.EncodeToString(checkpointParams.Checkpoint),
		AttestationTimestamp:   attestTimestamp2,
		LastConsensusTimestamp: attestTimestamp2,
		Signature:              signature2,
	}

	_, err = bridgeMsgServer.SubmitAttestationEvidence(ctx, evidenceMsg2)
	require.Error(err, "Second attestation evidence should fail due to rate limit")
	require.ErrorContains(err, "attestation evidence already submitted within rate limit")

	// try to submit third evidence outside rate limit window (should succeed)
	attestTimestamp3 := attestTimestamp1 + rateLimitMs + 1000 // 10+ minutes later
	maliciousValue3 := "0000000000000000000000000000000000000000000000000000000000000003"

	_, signature3 := s.createMaliciousAttestation(
		queryId,
		maliciousValue3,
		attestTimestamp3,
		checkpointParams.Checkpoint,
		targetPrivKey2, // use second validator for third attempt
	)

	evidenceMsg3 := &bridgetypes.MsgSubmitAttestationEvidence{
		Creator:                valAccAddrs[0].String(),
		QueryId:                queryId,
		Value:                  maliciousValue3,
		Timestamp:              attestTimestamp3,
		AggregatePower:         uint64(100),
		PreviousTimestamp:      attestTimestamp3 - 1000,
		NextTimestamp:          attestTimestamp3 + 1000,
		ValsetCheckpoint:       hex.EncodeToString(checkpointParams.Checkpoint),
		AttestationTimestamp:   attestTimestamp3,
		LastConsensusTimestamp: attestTimestamp3,
		Signature:              signature3,
	}

	_, err = bridgeMsgServer.SubmitAttestationEvidence(ctx, evidenceMsg3)
	require.NoError(err, "Third attestation evidence should succeed (outside rate limit)")
}

func (s *IntegrationTestSuite) TestAttestationSlashingUnbondingPeriod() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(30).WithBlockTime(time.Now())

	// create validator
	valAccAddrs, _, privKeys := s.createValidatorsWithEVMKeys(1, []uint64{1000})
	targetPrivKey := privKeys[0]

	// create old attestation (older than unbonding period)
	unbondingTime, err := s.Setup.Stakingkeeper.UnbondingTime(ctx)
	require.NoError(err)

	oldTime := ctx.BlockTime().Add(-unbondingTime).Add(-time.Hour) // 1 hour before unbonding period
	oldAttestTimestamp := uint64(oldTime.UnixMilli())

	// setup old checkpoint
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("old-checkpoint"),
		ValsetHash:     []byte("old-hash"),
		Timestamp:      oldAttestTimestamp,
		PowerThreshold: uint64(500),
		BlockHeight:    1, // old block
	}
	require.NoError(s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Set(ctx, oldAttestTimestamp, checkpointParams))

	queryId := "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4993"
	maliciousValue := "0000000000000000000000000000000000000000000000000000000000000001"

	_, signature := s.createMaliciousAttestation(
		queryId,
		maliciousValue,
		oldAttestTimestamp,
		checkpointParams.Checkpoint,
		targetPrivKey,
	)

	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg := &bridgetypes.MsgSubmitAttestationEvidence{
		Creator:                valAccAddrs[0].String(),
		QueryId:                queryId,
		Value:                  maliciousValue,
		Timestamp:              oldAttestTimestamp,
		AggregatePower:         uint64(100),
		PreviousTimestamp:      oldAttestTimestamp - 1000,
		NextTimestamp:          oldAttestTimestamp + 1000,
		ValsetCheckpoint:       hex.EncodeToString(checkpointParams.Checkpoint),
		AttestationTimestamp:   oldAttestTimestamp,
		LastConsensusTimestamp: oldAttestTimestamp,
		Signature:              signature,
	}

	_, err = bridgeMsgServer.SubmitAttestationEvidence(ctx, evidenceMsg)
	require.Error(err, "Old attestation evidence should be rejected")
	require.ErrorContains(err, "attestation timestamp is older than unbonding period")
}

// helper functions

type ValidatorState struct {
	Address sdk.ValAddress
	Tokens  math.Int
	Jailed  bool
	Status  stakingtypes.BondStatus
}

func (s *IntegrationTestSuite) createValidatorsWithEVMKeys(numValidators int, powers []uint64) ([]sdk.AccAddress, []sdk.ValAddress, []*ecdsa.PrivateKey) {
	ctx := s.Setup.Ctx

	// create ed25519 keys for validators
	privKeys := make([]ed25519.PrivKey, numValidators)
	for i := 0; i < numValidators; i++ {
		pk := ed25519.GenPrivKey()
		privKeys[i] = *pk
	}

	testAddrs := s.Setup.ConvertToAccAddress(privKeys)

	// mint tokens for validators
	for i, addr := range testAddrs {
		s.Setup.MintTokens(addr, math.NewInt(int64(powers[i]*1000000)))
	}

	valAddrs := simtestutil.ConvertAddrsToValAddrs(testAddrs)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)

	// create EVM private keys for signing
	evmPrivKeys := make([]*ecdsa.PrivateKey, numValidators)

	for i, pk := range privKeys {
		// create validator account
		account := authtypes.BaseAccount{
			Address:       testAddrs[i].String(),
			PubKey:        codectypes.UnsafePackAny(pk.PubKey()),
			AccountNumber: uint64(i + 200), // offset to avoid conflicts
		}
		s.Setup.Accountkeeper.SetAccount(ctx, &account)

		// create validator
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			valAddrs[i].String(),
			pk.PubKey(),
			sdk.NewInt64Coin(s.Setup.Denom, s.Setup.Stakingkeeper.TokensFromConsensusPower(ctx, int64(powers[i])).Int64()),
			stakingtypes.Description{Moniker: "val" + strconv.Itoa(i)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			math.OneInt())
		s.NoError(err)

		_, err = stakingServer.CreateValidator(ctx, valMsg)
		s.NoError(err)

		// create EVM private key for this validator
		evmPrivKey, err := crypto.GenerateKey()
		s.NoError(err)
		evmPrivKeys[i] = evmPrivKey

		// register EVM address for validator
		evmAddress := crypto.PubkeyToAddress(evmPrivKey.PublicKey)
		evmAddressBytes := evmAddress.Bytes()

		err = s.Setup.Bridgekeeper.EVMToOperatorAddressMap.Set(ctx,
			common.Bytes2Hex(evmAddressBytes),
			bridgetypes.OperatorAddress{OperatorAddress: valAddrs[i].Bytes()})
		s.NoError(err)

		err = s.Setup.Bridgekeeper.OperatorToEVMAddressMap.Set(ctx,
			valAddrs[i].String(),
			bridgetypes.EVMAddress{EVMAddress: evmAddressBytes})
		s.NoError(err)
	}

	// end block to finalize validator creation
	_, err := s.Setup.Stakingkeeper.EndBlocker(ctx)
	s.NoError(err)

	return testAddrs, valAddrs, evmPrivKeys
}

func (s *IntegrationTestSuite) createMaliciousAttestation(
	queryId string,
	value string,
	timestamp uint64,
	checkpoint []byte,
	privKey *ecdsa.PrivateKey,
) ([]byte, string) {
	require := s.Require()

	queryIdBytes, err := hex.DecodeString(queryId)
	require.NoError(err)

	// create snapshot bytes using the same encoding as the keeper
	snapshotBytes, err := s.Setup.Bridgekeeper.EncodeOracleAttestationData(
		queryIdBytes,
		value,
		timestamp,
		uint64(100),    // aggregate power
		timestamp-1000, // previous timestamp
		timestamp+1000, // next timestamp
		checkpoint,
		timestamp, // attestation timestamp
		timestamp, // last consensus timestamp
	)
	require.NoError(err)

	// sign the snapshot
	msgHash := sha256.Sum256(snapshotBytes)
	signature, err := crypto.Sign(msgHash[:], privKey)
	require.NoError(err)

	// remove recovery ID (bridge expects 64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)

	return snapshotBytes, sigHex
}
