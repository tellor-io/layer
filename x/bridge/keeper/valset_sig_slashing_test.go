package keeper_test

import (
	"encoding/hex"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"
)

func TestCheckValsetSignatureEvidence(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.SetValsetCheckpointDomainSeparator(sdkCtx)

	// Setup default parameters
	err := k.Params.Set(ctx, types.DefaultParams())
	require.NoError(t, err)

	// Test case: Penalty time cutoff
	t.Run("penalty_time_cutoff", func(t *testing.T) {
		// Setup unbonding period mock for this test
		unbondingTime := time.Hour * 24 * 14 // 14 days
		sk.On("UnbondingTime", ctx).Return(unbondingTime, nil)

		currentTime := ctx.BlockTime()
		cutoffTimestamp := currentTime.Add(-time.Hour * 24 * 7).UnixMilli()        // 7 days ago
		beforeCutoffTimestamp := currentTime.Add(-time.Hour * 24 * 10).UnixMilli() // 10 days ago (before cutoff)
		afterCutoffTimestamp := currentTime.Add(-time.Hour * 24 * 3).UnixMilli()   // 3 days ago (after cutoff)

		// Set the penalty time cutoff parameter
		params := types.DefaultParams()
		params.AttestPenaltyTimeCutoff = uint64(cutoffTimestamp)
		err := k.Params.Set(ctx, params)
		require.NoError(t, err)

		// Test case 1: Evidence before cutoff should be rejected
		beforeCutoffRequest := types.MsgSubmitValsetSignatureEvidence{
			Creator:            "creator",
			ValsetHash:         "1234abcd",
			ValidatorSignature: "0123456789abcdef",
			PowerThreshold:     2,
			ValsetTimestamp:    uint64(beforeCutoffTimestamp),
		}

		err = k.CheckValsetSignatureEvidence(ctx, beforeCutoffRequest)
		require.ErrorContains(t, err, "valset timestamp is before penalty cutoff")

		// Test case 2: Evidence after cutoff should pass cutoff check (but may fail other checks)
		afterCutoffRequest := types.MsgSubmitValsetSignatureEvidence{
			Creator:            "creator",
			ValsetHash:         "1234abcd",
			ValidatorSignature: "0123456789abcdef",
			PowerThreshold:     2,
			ValsetTimestamp:    uint64(afterCutoffTimestamp),
		}

		err = k.CheckValsetSignatureEvidence(ctx, afterCutoffRequest)
		// Should not fail with cutoff error, but may fail with other errors
		require.NotContains(t, err.Error(), "valset timestamp is before penalty cutoff")

		// Test case 3: When cutoff is 0 (disabled), evidence should pass cutoff check
		paramsNoCutoff := types.DefaultParams()
		paramsNoCutoff.AttestPenaltyTimeCutoff = 0
		err = k.Params.Set(ctx, paramsNoCutoff)
		require.NoError(t, err)

		err = k.CheckValsetSignatureEvidence(ctx, beforeCutoffRequest)
		// Should not fail with cutoff error when cutoff is disabled
		require.NotContains(t, err.Error(), "valset timestamp is before penalty cutoff")

		// Reset to default params for other tests
		err = k.Params.Set(ctx, types.DefaultParams())
		require.NoError(t, err)
	})

	// Test case 1: Reject evidence older than unbonding period
	// Setup unbonding period mock
	unbondingTime := time.Hour * 24 * 14 // 14 days
	sk.On("UnbondingTime", ctx).Return(unbondingTime, nil)

	currentTime := ctx.BlockTime()
	oldTimestamp := currentTime.Add(-time.Hour * 24 * 15).UnixMilli() // 15 days ago

	oldEvidenceRequest := types.MsgSubmitValsetSignatureEvidence{
		Creator:            "creator",
		ValsetHash:         "1234abcd",
		ValidatorSignature: "0123456789abcdef",
		PowerThreshold:     2,
		ValsetTimestamp:    uint64(oldTimestamp),
	}

	err = k.CheckValsetSignatureEvidence(ctx, oldEvidenceRequest)
	require.ErrorContains(t, err, "valset timestamp is older than unbonding period")

	// Test case 2: Reject evidence when checkpoint matches (not malicious)
	// Setup a new request with a valid timestamp
	recentTimestamp := currentTime.Add(-time.Hour * 24).UnixMilli() // 1 day ago

	validRequest := types.MsgSubmitValsetSignatureEvidence{
		Creator:            "creator",
		ValsetHash:         "1234abcd",
		ValidatorSignature: "0123456789abcdef",
		PowerThreshold:     2,
		ValsetTimestamp:    uint64(recentTimestamp),
	}

	// Set up the valset in storage
	err = k.BridgeValsetByTimestampMap.Set(ctx, validRequest.ValsetTimestamp, types.BridgeValidatorSet{})
	require.NoError(t, err)

	// Set up matching checkpoint params
	valsetHashBytes, err := hex.DecodeString(validRequest.ValsetHash)
	require.NoError(t, err)
	checkpoint, err := k.EncodeValsetCheckpoint(ctx, validRequest.PowerThreshold, validRequest.ValsetTimestamp, valsetHashBytes)
	require.NoError(t, err)

	err = k.ValidatorCheckpointParamsMap.Set(ctx, validRequest.ValsetTimestamp, types.ValidatorCheckpointParams{
		Checkpoint:     checkpoint,
		BlockHeight:    100,
		Timestamp:      validRequest.ValsetTimestamp,
		PowerThreshold: validRequest.PowerThreshold,
	})
	require.NoError(t, err)

	// This should fail because the checkpoint matches (meaning it's not malicious)
	err = k.CheckValsetSignatureEvidence(ctx, validRequest)
	require.ErrorContains(t, err, "checkpoint matches the actual checkpoint, not malicious")

	// Test case 3: Reject evidence with invalid signature
	invalidSigRequest := types.MsgSubmitValsetSignatureEvidence{
		Creator:            "creator",
		ValsetHash:         "1234abcd",
		ValidatorSignature: "invalidhex",
		PowerThreshold:     3,
		ValsetTimestamp:    uint64(recentTimestamp),
	}

	err = k.CheckValsetSignatureEvidence(ctx, invalidSigRequest)
	require.ErrorContains(t, err, "encoding/hex")
}

func TestCheckValsetSignatureRateLimit(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.SetValsetCheckpointDomainSeparator(sdkCtx)

	// Setup default parameters
	err := k.Params.Set(ctx, types.DefaultParams())
	require.NoError(t, err)

	// get the actual rate limit window from the keeper
	rateLimitMs, err := k.GetValsetRateLimitWindow(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(10*60*1000), rateLimitMs) // should be 10 minutes default

	// create a mock operator address
	operatorAddr := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper1test"),
	}

	// test timestamps
	baseTime := uint64(1000000000) // arbitrary base timestamp

	// Test case 1: First submission should succeed (no rate limiting yet)
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, baseTime),
		true)
	require.NoError(t, err)

	// Test case 2: Exact same timestamp should fail (duplicate)
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, baseTime)
	require.ErrorContains(t, err, "valset signature evidence already submitted with this timestamp")

	// Test case 3: Timestamp within rate limit window (after baseTime) should fail
	withinWindowAfter := baseTime + (rateLimitMs / 2) // 5 minutes after
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, withinWindowAfter)
	require.ErrorContains(t, err, "valset signature evidence already submitted within rate limit")

	// Test case 4: Timestamp within rate limit window (before baseTime) should fail
	withinWindowBefore := baseTime - (rateLimitMs / 2) // 5 minutes before
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, withinWindowBefore)
	require.ErrorContains(t, err, "valset signature evidence already submitted within rate limit")

	// Test case 5: Timestamp outside rate limit window (after) should succeed
	outsideWindowAfter := baseTime + rateLimitMs + 1000 // 10+ minutes after
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, outsideWindowAfter)
	require.NoError(t, err)

	// Test case 6: Timestamp outside rate limit window (before) should succeed
	outsideWindowBefore := baseTime - rateLimitMs - 1000 // 10+ minutes before
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, outsideWindowBefore)
	require.NoError(t, err)

	// Test case 7: Different operator should not be affected by rate limit
	differentOperator := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper2test"),
	}
	err = k.CheckValsetSignatureRateLimit(ctx, differentOperator, baseTime)
	require.NoError(t, err)

	// Test case 8: Multiple submissions outside window should create new rate limit boundaries
	// Submit evidence at outsideWindowAfter
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, outsideWindowAfter),
		true)
	require.NoError(t, err)

	// Now test rate limiting around the new timestamp
	withinNewWindow := outsideWindowAfter + (rateLimitMs / 2)
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, withinNewWindow)
	require.ErrorContains(t, err, "valset signature evidence already submitted within rate limit")

	// But outside the new window should still work
	outsideNewWindow := outsideWindowAfter + rateLimitMs + 1000
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, outsideNewWindow)
	require.NoError(t, err)

	// Test case 9: test rate limit parameter retrieval error handling
	// This would be hard to test directly since GetValsetRateLimitWindow uses params
	// but we can at least verify it behaves consistently
	rateLimitMs2, err := k.GetValsetRateLimitWindow(ctx)
	require.NoError(t, err)
	require.Equal(t, rateLimitMs, rateLimitMs2)
}

func TestGetValsetEvidenceSubmittedBefore(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	operatorAddr := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper1test"),
	}

	// Test case 1: No evidence submitted before timestamp
	submitted, timestamp, err := k.GetValsetEvidenceSubmittedBefore(ctx, operatorAddr, uint64(1000))
	require.Error(t, err)
	require.False(t, submitted)
	require.Equal(t, uint64(0), timestamp)
	require.ErrorContains(t, err, "no valset evidence submitted before timestamp")

	// Test case 2: Add evidence and test retrieval
	baseTime := uint64(1000000000)
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, baseTime),
		true)
	require.NoError(t, err)

	// Search before the baseTime - should find nothing
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedBefore(ctx, operatorAddr, baseTime)
	require.Error(t, err)
	require.False(t, submitted)
	require.Equal(t, uint64(0), timestamp)

	// Search after the baseTime - should find the evidence
	searchTime := baseTime + 1000
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedBefore(ctx, operatorAddr, searchTime)
	require.NoError(t, err)
	require.True(t, submitted)
	require.Equal(t, baseTime, timestamp)

	// Test case 3: Multiple evidence submissions - should return most recent
	earlierTime := baseTime - 5000
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, earlierTime),
		true)
	require.NoError(t, err)

	// Should return the most recent (baseTime), not the earlier one
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedBefore(ctx, operatorAddr, searchTime)
	require.NoError(t, err)
	require.True(t, submitted)
	require.Equal(t, baseTime, timestamp)

	// Test case 4: Different operator should not interfere
	differentOperator := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper2test"),
	}
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedBefore(ctx, differentOperator, searchTime)
	require.Error(t, err)
	require.False(t, submitted)
	require.Equal(t, uint64(0), timestamp)
}

func TestGetValsetEvidenceSubmittedAfter(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	operatorAddr := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper1test"),
	}

	// Test case 1: No evidence submitted after timestamp
	submitted, timestamp, err := k.GetValsetEvidenceSubmittedAfter(ctx, operatorAddr, uint64(1000))
	require.Error(t, err)
	require.False(t, submitted)
	require.Equal(t, uint64(0), timestamp)
	require.ErrorContains(t, err, "no valset evidence submitted after timestamp")

	// Test case 2: Add evidence and test retrieval
	baseTime := uint64(1000000000)
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, baseTime),
		true)
	require.NoError(t, err)

	// Search after the baseTime - should find nothing
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedAfter(ctx, operatorAddr, baseTime)
	require.Error(t, err)
	require.False(t, submitted)
	require.Equal(t, uint64(0), timestamp)

	// Search before the baseTime - should find the evidence
	searchTime := baseTime - 1000
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedAfter(ctx, operatorAddr, searchTime)
	require.NoError(t, err)
	require.True(t, submitted)
	require.Equal(t, baseTime, timestamp)

	// Test case 3: Multiple evidence submissions - should return earliest
	laterTime := baseTime + 5000
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, laterTime),
		true)
	require.NoError(t, err)

	// Should return the earliest after search time (baseTime), not the later one
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedAfter(ctx, operatorAddr, searchTime)
	require.NoError(t, err)
	require.True(t, submitted)
	require.Equal(t, baseTime, timestamp)

	// Test case 4: Different operator should not interfere
	differentOperator := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper2test"),
	}
	submitted, timestamp, err = k.GetValsetEvidenceSubmittedAfter(ctx, differentOperator, searchTime)
	require.Error(t, err)
	require.False(t, submitted)
	require.Equal(t, uint64(0), timestamp)
}

func TestGetCheckpointParamsBefore(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// Test case 1: No checkpoint params before timestamp
	_, err := k.GetCheckpointParamsBefore(ctx, uint64(1000))
	require.Error(t, err)
	require.ErrorContains(t, err, "no checkpoint params found before timestamp")

	// Test case 2: Add checkpoint params and test retrieval
	baseTime := uint64(1000000000)
	checkpointParams := types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint1"),
		ValsetHash:     []byte("valsetHash1"),
		Timestamp:      baseTime,
		PowerThreshold: uint64(66),
		BlockHeight:    100,
	}

	err = k.ValidatorCheckpointParamsMap.Set(ctx, baseTime, checkpointParams)
	require.NoError(t, err)

	// Search before the baseTime - should find nothing
	_, err = k.GetCheckpointParamsBefore(ctx, baseTime)
	require.Error(t, err)
	require.ErrorContains(t, err, "no checkpoint params found before timestamp")

	// Search after the baseTime - should find the checkpoint params
	searchTime := baseTime + 1000
	result, err := k.GetCheckpointParamsBefore(ctx, searchTime)
	require.NoError(t, err)
	require.Equal(t, checkpointParams.Checkpoint, result.Checkpoint)
	require.Equal(t, checkpointParams.ValsetHash, result.ValsetHash)
	require.Equal(t, checkpointParams.Timestamp, result.Timestamp)
	require.Equal(t, checkpointParams.PowerThreshold, result.PowerThreshold)
	require.Equal(t, checkpointParams.BlockHeight, result.BlockHeight)

	// Test case 3: Multiple checkpoint params - should return most recent
	earlierTime := baseTime - 5000
	earlierCheckpointParams := types.ValidatorCheckpointParams{
		Checkpoint:     []byte("checkpoint0"),
		ValsetHash:     []byte("valsetHash0"),
		Timestamp:      earlierTime,
		PowerThreshold: uint64(50),
		BlockHeight:    50,
	}

	err = k.ValidatorCheckpointParamsMap.Set(ctx, earlierTime, earlierCheckpointParams)
	require.NoError(t, err)

	// Should return the most recent (baseTime), not the earlier one
	result, err = k.GetCheckpointParamsBefore(ctx, searchTime)
	require.NoError(t, err)
	require.Equal(t, checkpointParams.Checkpoint, result.Checkpoint)
	require.Equal(t, baseTime, result.Timestamp)

	// Test case 4: Search between two checkpoints
	middleSearchTime := baseTime - 1000
	result, err = k.GetCheckpointParamsBefore(ctx, middleSearchTime)
	require.NoError(t, err)
	require.Equal(t, earlierCheckpointParams.Checkpoint, result.Checkpoint)
	require.Equal(t, earlierTime, result.Timestamp)

	// Test case 5: Edge case - exact timestamp match should not be included
	result, err = k.GetCheckpointParamsBefore(ctx, baseTime)
	require.NoError(t, err)
	require.Equal(t, earlierCheckpointParams.Checkpoint, result.Checkpoint)
	require.Equal(t, earlierTime, result.Timestamp)
}
