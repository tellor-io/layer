package integration_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	bridgekeeper "github.com/tellor-io/layer/x/bridge/keeper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"
)

func (s *IntegrationTestSuite) TestValsetSignatureSlashingIntegration() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(10).WithBlockTime(time.Now())

	// create validators with real staking
	numValidators := 3
	validatorPowers := []uint64{1000, 2000, 3000} // different powers for testing

	valAccAddrs, valAddrs, privKeys := s.createValidatorsWithEVMKeys(numValidators, validatorPowers)

	// setup bridge validator checkpoints for historical reference
	startTime := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("historical-checkpoint-123"),
		ValsetHash:     []byte("historical-valset-hash"),
		Timestamp:      startTime,
		PowerThreshold: uint64(4000), // 2/3 of total power (6000)
		BlockHeight:    5,            // earlier block
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

	// Test case 1: Submit valid valset signature evidence and verify slashing
	targetValidatorIndex := 1 // slash the middle validator
	targetValidator := valAddrs[targetValidatorIndex]
	targetPrivKey := privKeys[targetValidatorIndex]

	// create malicious valset signature data
	valsetTimestamp := uint64(time.Now().UnixMilli())
	fakeValsetHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	powerThreshold := uint64(5000) // fake power threshold

	// create and sign malicious valset checkpoint
	maliciousCheckpoint, signature := s.createMaliciousValsetSignature(
		powerThreshold,
		valsetTimestamp,
		fakeValsetHash,
		targetPrivKey,
	)

	// verify this valset doesn't exist or has different checkpoint (making it malicious)
	exists, err := s.Setup.Bridgekeeper.BridgeValsetByTimestampMap.Has(ctx, valsetTimestamp)
	require.NoError(err)
	if exists {
		// if it exists, make sure the checkpoint is different
		existingParams, err := s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Get(ctx, valsetTimestamp)
		require.NoError(err)
		require.False(bytes.Equal(maliciousCheckpoint, existingParams.Checkpoint),
			"Checkpoint should be different to be considered malicious")
	}

	// submit valset signature evidence
	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg := &bridgetypes.MsgSubmitValsetSignatureEvidence{
		Creator:            valAccAddrs[0].String(), // submitter
		ValsetTimestamp:    valsetTimestamp,
		ValsetHash:         fakeValsetHash,
		PowerThreshold:     powerThreshold,
		ValidatorSignature: signature,
	}

	_, err = bridgeMsgServer.SubmitValsetSignatureEvidence(ctx, evidenceMsg)
	require.NoError(err, "Valset signature evidence submission should succeed")

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
	evidenceKey := collections.Join(targetValidator.Bytes(), valsetTimestamp)
	evidenceExists, err := s.Setup.Bridgekeeper.ValsetSignatureEvidenceSubmitted.Has(ctx, evidenceKey)
	require.NoError(err)
	require.True(evidenceExists, "Evidence should be recorded")

	evidenceValue, err := s.Setup.Bridgekeeper.ValsetSignatureEvidenceSubmitted.Get(ctx, evidenceKey)
	require.NoError(err)
	require.True(evidenceValue.Submitted, "Evidence should be marked as submitted")
}

func (s *IntegrationTestSuite) TestValsetSignatureSlashingRateLimit() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(20).WithBlockTime(time.Now())

	// create multiple validators for rate limit testing (since first one gets jailed)
	valAccAddrs, valAddrs, privKeys := s.createValidatorsWithEVMKeys(2, []uint64{1000, 1000})
	targetValidator1 := valAddrs[0]
	targetValidator2 := valAddrs[1]
	targetPrivKey1 := privKeys[0]
	targetPrivKey2 := privKeys[1]

	// setup historical checkpoint for slashing reference
	startTime := uint64(time.Now().Add(-1 * time.Hour).UnixMilli())
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("rate-limit-checkpoint"),
		ValsetHash:     []byte("rate-limit-hash"),
		Timestamp:      startTime,
		PowerThreshold: uint64(1500),
		BlockHeight:    15,
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

	// submit first valset signature evidence (using first validator)
	valsetTimestamp1 := uint64(time.Now().UnixMilli())
	fakeValsetHash1 := "1111111111111111111111111111111111111111111111111111111111111111"
	powerThreshold := uint64(1500)

	_, signature1 := s.createMaliciousValsetSignature(
		powerThreshold,
		valsetTimestamp1,
		fakeValsetHash1,
		targetPrivKey1,
	)

	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg1 := &bridgetypes.MsgSubmitValsetSignatureEvidence{
		Creator:            valAccAddrs[0].String(),
		ValsetTimestamp:    valsetTimestamp1,
		ValsetHash:         fakeValsetHash1,
		PowerThreshold:     powerThreshold,
		ValidatorSignature: signature1,
	}

	_, err = bridgeMsgServer.SubmitValsetSignatureEvidence(ctx, evidenceMsg1)
	require.NoError(err, "First valset signature evidence should succeed")

	// try to submit second evidence within rate limit window (should fail)
	rateLimitMs, err := s.Setup.Bridgekeeper.GetValsetRateLimitWindow(ctx)
	require.NoError(err)
	require.Equal(uint64(10*60*1000), rateLimitMs) // 10 minutes default

	valsetTimestamp2 := valsetTimestamp1 + (rateLimitMs / 2) // 5 minutes later
	fakeValsetHash2 := "2222222222222222222222222222222222222222222222222222222222222222"

	_, signature2 := s.createMaliciousValsetSignature(
		powerThreshold,
		valsetTimestamp2,
		fakeValsetHash2,
		targetPrivKey1, // same validator for rate limit test
	)

	evidenceMsg2 := &bridgetypes.MsgSubmitValsetSignatureEvidence{
		Creator:            valAccAddrs[0].String(),
		ValsetTimestamp:    valsetTimestamp2,
		ValsetHash:         fakeValsetHash2,
		PowerThreshold:     powerThreshold,
		ValidatorSignature: signature2,
	}

	_, err = bridgeMsgServer.SubmitValsetSignatureEvidence(ctx, evidenceMsg2)
	require.Error(err, "Second valset signature evidence should fail due to rate limit")
	require.ErrorContains(err, "valset signature evidence already submitted within rate limit")

	// try to submit third evidence outside rate limit window (should succeed)
	valsetTimestamp3 := valsetTimestamp1 + rateLimitMs + 1000 // 10+ minutes later
	fakeValsetHash3 := "3333333333333333333333333333333333333333333333333333333333333333"

	_, signature3 := s.createMaliciousValsetSignature(
		powerThreshold,
		valsetTimestamp3,
		fakeValsetHash3,
		targetPrivKey2, // use second validator for third attempt
	)

	evidenceMsg3 := &bridgetypes.MsgSubmitValsetSignatureEvidence{
		Creator:            valAccAddrs[0].String(),
		ValsetTimestamp:    valsetTimestamp3,
		ValsetHash:         fakeValsetHash3,
		PowerThreshold:     powerThreshold,
		ValidatorSignature: signature3,
	}

	_, err = bridgeMsgServer.SubmitValsetSignatureEvidence(ctx, evidenceMsg3)
	require.NoError(err, "Third valset signature evidence should succeed (outside rate limit)")
}

func (s *IntegrationTestSuite) TestValsetSignatureSlashingUnbondingPeriod() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(30).WithBlockTime(time.Now())

	// create validator
	valAccAddrs, _, privKeys := s.createValidatorsWithEVMKeys(1, []uint64{1000})
	targetPrivKey := privKeys[0]

	// create old valset signature (older than unbonding period)
	unbondingTime, err := s.Setup.Stakingkeeper.UnbondingTime(ctx)
	require.NoError(err)

	oldTime := ctx.BlockTime().Add(-unbondingTime).Add(-time.Hour) // 1 hour before unbonding period
	oldValsetTimestamp := uint64(oldTime.UnixMilli())

	// setup old checkpoint for historical reference
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     []byte("old-valset-checkpoint"),
		ValsetHash:     []byte("old-valset-hash"),
		Timestamp:      oldValsetTimestamp - 1000, // even older for reference
		PowerThreshold: uint64(500),
		BlockHeight:    1, // old block
	}
	require.NoError(s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Set(ctx, oldValsetTimestamp-1000, checkpointParams))

	fakeValsetHash := "4444444444444444444444444444444444444444444444444444444444444444"
	powerThreshold := uint64(500)

	_, signature := s.createMaliciousValsetSignature(
		powerThreshold,
		oldValsetTimestamp,
		fakeValsetHash,
		targetPrivKey,
	)

	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg := &bridgetypes.MsgSubmitValsetSignatureEvidence{
		Creator:            valAccAddrs[0].String(),
		ValsetTimestamp:    oldValsetTimestamp,
		ValsetHash:         fakeValsetHash,
		PowerThreshold:     powerThreshold,
		ValidatorSignature: signature,
	}

	_, err = bridgeMsgServer.SubmitValsetSignatureEvidence(ctx, evidenceMsg)
	require.Error(err, "Old valset signature evidence should be rejected")
	require.ErrorContains(err, "valset timestamp is older than unbonding period")
}

func (s *IntegrationTestSuite) TestValsetSignatureSlashingNonMalicious() {
	require := s.Require()
	ctx := s.Setup.Ctx.WithBlockHeight(40).WithBlockTime(time.Now())

	// create validator
	valAccAddrs, valAddrs, privKeys := s.createValidatorsWithEVMKeys(1, []uint64{1000})
	targetValidator := valAddrs[0]
	targetPrivKey := privKeys[0]

	// create a legitimate valset signature (should not be slashed)
	valsetTimestamp := uint64(time.Now().UnixMilli())
	realValsetHash := "5555555555555555555555555555555555555555555555555555555555555555"
	powerThreshold := uint64(1000)

	// create the real checkpoint and store it
	realCheckpoint, signature := s.createMaliciousValsetSignature(
		powerThreshold,
		valsetTimestamp,
		realValsetHash,
		targetPrivKey,
	)

	// store the real valset and checkpoint (making it legitimate)
	evmAddr, err := s.Setup.Bridgekeeper.OperatorToEVMAddressMap.Get(ctx, targetValidator.String())
	require.NoError(err)

	bridgeValset := bridgetypes.BridgeValidatorSet{
		BridgeValidatorSet: []*bridgetypes.BridgeValidator{
			{
				EthereumAddress: evmAddr.EVMAddress,
				Power:           1000,
			},
		},
	}
	require.NoError(s.Setup.Bridgekeeper.BridgeValsetByTimestampMap.Set(ctx, valsetTimestamp, bridgeValset))

	// store the matching checkpoint params
	checkpointParams := bridgetypes.ValidatorCheckpointParams{
		Checkpoint:     realCheckpoint,
		ValsetHash:     []byte(realValsetHash),
		Timestamp:      valsetTimestamp,
		PowerThreshold: powerThreshold,
		BlockHeight:    40,
	}
	require.NoError(s.Setup.Bridgekeeper.ValidatorCheckpointParamsMap.Set(ctx, valsetTimestamp, checkpointParams))

	bridgeMsgServer := bridgekeeper.NewMsgServerImpl(s.Setup.Bridgekeeper)
	evidenceMsg := &bridgetypes.MsgSubmitValsetSignatureEvidence{
		Creator:            valAccAddrs[0].String(),
		ValsetTimestamp:    valsetTimestamp,
		ValsetHash:         realValsetHash,
		PowerThreshold:     powerThreshold,
		ValidatorSignature: signature,
	}

	_, err = bridgeMsgServer.SubmitValsetSignatureEvidence(ctx, evidenceMsg)
	require.Error(err, "Non-malicious valset signature evidence should be rejected")
	require.ErrorContains(err, "checkpoint matches the actual checkpoint, not malicious")
}

// helper functions

func (s *IntegrationTestSuite) createMaliciousValsetSignature(
	powerThreshold uint64,
	valsetTimestamp uint64,
	valsetHash string,
	privKey *ecdsa.PrivateKey,
) ([]byte, string) {
	require := s.Require()

	// create valset checkpoint using the same encoding as the keeper
	checkpoint, err := s.encodeValsetCheckpoint(powerThreshold, valsetTimestamp, valsetHash)
	require.NoError(err)

	// sign the checkpoint
	msgHash := sha256.Sum256(checkpoint)
	signature, err := crypto.Sign(msgHash[:], privKey)
	require.NoError(err)

	// remove recovery ID (bridge expects 64 bytes)
	signature = signature[:64]
	sigHex := hex.EncodeToString(signature)

	return checkpoint, sigHex
}

func (s *IntegrationTestSuite) encodeValsetCheckpoint(powerThreshold, valsetTimestamp uint64, valsetHash string) ([]byte, error) {
	// define the domain separator for the validator set hash, fixed size 32 bytes
	VALIDATOR_SET_HASH_DOMAIN_SEPARATOR := []byte("checkpoint")
	var domainSeparatorFixSize [32]byte
	copy(domainSeparatorFixSize[:], VALIDATOR_SET_HASH_DOMAIN_SEPARATOR)

	// convert valsetHash to bytes
	valsetHashBytes, err := hex.DecodeString(valsetHash)
	if err != nil {
		return nil, err
	}

	// convert valsetHash to a fixed size 32 bytes
	var valsetHashFixSize [32]byte
	copy(valsetHashFixSize[:], valsetHashBytes)

	// convert powerThreshold and valsetTimestamp to *big.Int for ABI encoding
	powerThresholdBigInt := new(big.Int).SetUint64(powerThreshold)
	valsetTimestampBigInt := new(big.Int).SetUint64(valsetTimestamp)

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	// prepare the types for encoding
	arguments := abi.Arguments{
		{Type: bytes32Type},
		{Type: uint256Type},
		{Type: uint256Type},
		{Type: bytes32Type},
	}

	// encode the arguments
	encodedCheckpointData, err := arguments.Pack(
		domainSeparatorFixSize,
		powerThresholdBigInt,
		valsetTimestampBigInt,
		valsetHashFixSize,
	)
	if err != nil {
		return nil, err
	}

	checkpoint := crypto.Keccak256(encodedCheckpointData)
	return checkpoint, nil
}
