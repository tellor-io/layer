package keeper_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestCheckValsetSignatureEvidence(t *testing.T) {
	k, _, _, _, _, sk, _, ctx := testkeeper.BridgeKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

	// Setup default parameters
	err := k.Params.Set(ctx, types.DefaultParams())
	require.NoError(t, err)

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

	// Test case 4: Rate limiting functionality
	testRateLimiting(t, k, ctx)
}

func testRateLimiting(t *testing.T, k keeper.Keeper, ctx context.Context) {
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

	// Test case 4a: First submission should succeed (no rate limiting yet)
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, baseTime),
		types.BoolSubmitted{Submitted: true})
	require.NoError(t, err)

	// Test case 4b: Exact same timestamp should fail (duplicate)
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, baseTime)
	require.ErrorContains(t, err, "valset signature evidence already submitted with this timestamp")

	// Test case 4c: Timestamp within rate limit window (after baseTime) should fail
	withinWindowAfter := baseTime + (rateLimitMs / 2) // 5 minutes after
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, withinWindowAfter)
	require.ErrorContains(t, err, "valset signature evidence already submitted within rate limit")

	// Test case 4d: Timestamp within rate limit window (before baseTime) should fail
	withinWindowBefore := baseTime - (rateLimitMs / 2) // 5 minutes before
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, withinWindowBefore)
	require.ErrorContains(t, err, "valset signature evidence already submitted within rate limit")

	// Test case 4e: Timestamp outside rate limit window (after) should succeed
	outsideWindowAfter := baseTime + rateLimitMs + 1000 // 10+ minutes after
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, outsideWindowAfter)
	require.NoError(t, err)

	// Test case 4f: Timestamp outside rate limit window (before) should succeed
	outsideWindowBefore := baseTime - rateLimitMs - 1000 // 10+ minutes before
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, outsideWindowBefore)
	require.NoError(t, err)

	// Test case 4g: Different operator should not be affected by rate limit
	differentOperator := types.OperatorAddress{
		OperatorAddress: []byte("cosmosvaloper2test"),
	}
	err = k.CheckValsetSignatureRateLimit(ctx, differentOperator, baseTime)
	require.NoError(t, err)

	// Test case 4h: Multiple submissions outside window should create new rate limit boundaries
	// Submit evidence at outsideWindowAfter
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx,
		collections.Join(operatorAddr.OperatorAddress, outsideWindowAfter),
		types.BoolSubmitted{Submitted: true})
	require.NoError(t, err)

	// Now test rate limiting around the new timestamp
	withinNewWindow := outsideWindowAfter + (rateLimitMs / 2)
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, withinNewWindow)
	require.ErrorContains(t, err, "valset signature evidence already submitted within rate limit")

	// But outside the new window should still work
	outsideNewWindow := outsideWindowAfter + rateLimitMs + 1000
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, outsideNewWindow)
	require.NoError(t, err)
}
