package keeper_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
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
		PowerThreshold:     2,
		ValsetTimestamp:    uint64(recentTimestamp),
	}

	err = k.CheckValsetSignatureEvidence(ctx, invalidSigRequest)
	require.ErrorContains(t, err, "encoding/hex")

	// For the remaining test cases, we would need more complex mocking and setup
	// that might be challenging without deeper knowledge of the codebase internals.
	// The test can be expanded when we have a better understanding of how to mock:
	// 1. The rate limiting functionality
	// 2. The validator signature verification process
	// 3. The validator slashing process

	t.Log("Additional test cases for rate limiting, signature verification, and slashing should be implemented with better understanding of module internals")
}
