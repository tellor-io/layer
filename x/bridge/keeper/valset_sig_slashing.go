package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CheckValsetSignatureEvidence checks whether malicious validator set signature evidence is valid and should be slashed.
// If it is valid, it will slash the validator.
func (k Keeper) CheckValsetSignatureEvidence(ctx context.Context, request types.MsgSubmitValsetSignatureEvidence) error {
	// check whether valset timestamp is older than unbonding period.
	// if it is, we can return an error
	unbondingPeriod, err := k.stakingKeeper.UnbondingTime(ctx)
	if err != nil {
		return err
	}
	currentTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	unbondingTime := currentTime.Add(-unbondingPeriod)
	if request.ValsetTimestamp < uint64(unbondingTime.UnixMilli()) {
		return errors.New("valset timestamp is older than unbonding period")
	}

	// get the checkpoint from the inputted params
	valsetHashBytes, err := hex.DecodeString(request.ValsetHash)
	if err != nil {
		return err
	}
	checkpoint, err := k.EncodeValsetCheckpoint(ctx, request.PowerThreshold, request.ValsetTimestamp, valsetHashBytes)
	if err != nil {
		return err
	}

	// check whether a validator set with this timestamp exists, and the checkpoints match
	exists, err := k.BridgeValsetByTimestampMap.Has(ctx, request.ValsetTimestamp)
	if err != nil {
		return err
	}
	if exists {
		// get the checkpoint params and check if the checkpoints match. if they do, this is not malicious
		checkpointParams, err := k.ValidatorCheckpointParamsMap.Get(ctx, request.ValsetTimestamp)
		if err != nil {
			return err
		}
		if bytes.Equal(checkpoint, checkpointParams.Checkpoint) {
			return errors.New("checkpoint matches the actual checkpoint, not malicious")
		}
	}

	// verify the signature corresponds to a valid operator address
	operatorAddr, err := k.GetOperatorAddressFromSignature(ctx, checkpoint, request.ValidatorSignature)
	if err != nil {
		return err
	}

	if operatorAddr.OperatorAddress == nil {
		return errors.New("operator address not found")
	}

	// check the rate limit
	err = k.CheckValsetSignatureRateLimit(ctx, operatorAddr, request.ValsetTimestamp)
	if err != nil {
		return err
	}

	// slash the validator
	slashFactor, err := k.GetValsetSlashPercentage(ctx)
	if err != nil {
		return err
	}

	checkpointParamsBefore, err := k.GetCheckpointParamsBefore(ctx, request.ValsetTimestamp)
	if err != nil {
		return err
	}

	err = k.SlashValidator(ctx, operatorAddr, slashFactor, checkpointParamsBefore.BlockHeight)
	if err != nil {
		return err
	}

	// record that valset signature evidence was submitted
	err = k.ValsetSignatureEvidenceSubmitted.Set(ctx, collections.Join(operatorAddr.OperatorAddress, request.ValsetTimestamp), types.BoolSubmitted{
		Submitted: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// CheckValsetSignatureRateLimit checks whether valset signature evidence has been submitted for a given operator address
// with a valset timestamp that is within the rate limit
func (k Keeper) CheckValsetSignatureRateLimit(ctx context.Context, operatorAddr types.OperatorAddress, valsetTimestamp uint64) error {
	rateLimit, err := k.GetValsetRateLimitWindow(ctx)
	if err != nil {
		return err
	}

	// first, check whether there is already a submission with this exact timestamp
	submitted, err := k.ValsetSignatureEvidenceSubmitted.Has(ctx, collections.Join(operatorAddr.OperatorAddress, valsetTimestamp))
	if err != nil {
		return err
	}
	if submitted {
		return errors.New("valset signature evidence already submitted with this timestamp")
	}

	// check for evidence submitted before the valset timestamp
	submitted, timestampBefore, _ := k.GetValsetEvidenceSubmittedBefore(ctx, operatorAddr, valsetTimestamp)
	if submitted && valsetTimestamp >= timestampBefore {
		// check if the timestamp is within the rate limit
		if valsetTimestamp-timestampBefore < rateLimit {
			return errors.New("valset signature evidence already submitted within rate limit")
		}
	}

	// check for evidence submitted after the valset timestamp
	submitted, timestampAfter, _ := k.GetValsetEvidenceSubmittedAfter(ctx, operatorAddr, valsetTimestamp)
	if submitted && timestampAfter >= valsetTimestamp {
		// check if the timestamp is within the rate limit
		if timestampAfter-valsetTimestamp < rateLimit {
			return errors.New("valset signature evidence already submitted within rate limit")
		}
	}

	return nil
}

// GetValsetEvidenceSubmittedBefore returns the timestamp of any valset signature evidence submitted
// for a given operator address before a given timestamp
func (k Keeper) GetValsetEvidenceSubmittedBefore(ctx context.Context, operatorAddress types.OperatorAddress, timestamp uint64) (submitted bool, timestampBefore uint64, err error) {
	// create a range that ends just before this timestamp
	rng := collections.NewPrefixedPairRange[[]byte, uint64](operatorAddress.OperatorAddress).EndExclusive(timestamp).Descending()

	var mostRecent bool
	var mostRecentTimestamp uint64

	// walk through the submissions in descending order to find the most recent one before the timestamp
	err = k.ValsetSignatureEvidenceSubmitted.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.BoolSubmitted) (stop bool, err error) {
		mostRecent = value.Submitted
		mostRecentTimestamp = key.K2()
		return true, nil // stop after the first (most recent) match
	})
	if err != nil {
		return false, 0, err
	}

	if !mostRecent {
		return false, 0, fmt.Errorf("no valset evidence submitted before timestamp %v for operator address %s", timestamp, operatorAddress.String())
	}

	return mostRecent, mostRecentTimestamp, nil
}

// GetValsetEvidenceSubmittedAfter returns the timestamp of any valset signature evidence submitted
// for a given operator address after a given timestamp
func (k Keeper) GetValsetEvidenceSubmittedAfter(ctx context.Context, operatorAddr types.OperatorAddress, timestamp uint64) (bool, uint64, error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](operatorAddr.OperatorAddress).StartExclusive(timestamp)

	var oldest bool
	var oldestTimestamp uint64

	err := k.ValsetSignatureEvidenceSubmitted.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.BoolSubmitted) (stop bool, err error) {
		oldest = value.Submitted
		oldestTimestamp = key.K2()
		return true, nil
	})
	if err != nil {
		return false, 0, err
	}

	if !oldest {
		return false, 0, fmt.Errorf("no valset evidence submitted after timestamp %v for operator address %s", timestamp, operatorAddr.String())
	}

	return oldest, oldestTimestamp, nil
}

// GetCheckpointParamsBefore returns the validator checkpoint params with the highest timestamp
// that is before the specified timestamp
func (k Keeper) GetCheckpointParamsBefore(ctx context.Context, timestamp uint64) (types.ValidatorCheckpointParams, error) {
	// create a range that ends just before this timestamp and is in descending order
	// to get the most recent checkpoint before the given timestamp
	rng := new(collections.Range[uint64]).EndExclusive(timestamp).Descending()

	var checkpointParams types.ValidatorCheckpointParams
	var found bool

	// walk through the checkpoint params in descending order to find the most recent one
	err := k.ValidatorCheckpointParamsMap.Walk(ctx, rng, func(key uint64, value types.ValidatorCheckpointParams) (bool, error) {
		checkpointParams = value
		found = true
		return true, nil // stop after the first (most recent) match
	})

	if err != nil {
		return types.ValidatorCheckpointParams{}, err
	}

	if !found {
		return types.ValidatorCheckpointParams{}, fmt.Errorf("no checkpoint params found before timestamp %v", timestamp)
	}

	return checkpointParams, nil
}
