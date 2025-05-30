package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	layerconfig "github.com/tellor-io/layer/app/config"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

// CheckAttestationEvidence checks whether malicious attestation evidence is valid and should be slashed. If it is, it will slash
// the validator.
func (k Keeper) CheckAttestationEvidence(ctx context.Context, request types.MsgSubmitAttestationEvidence) error {
	// check whether attestation timestamp is older than unbonding period.
	// if it is, we can return an error
	unbondingPeriod, err := k.stakingKeeper.UnbondingTime(ctx)
	if err != nil {
		return err
	}
	currentTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	unbondingTime := currentTime.Add(-unbondingPeriod)
	if request.AttestationTimestamp < uint64(unbondingTime.UnixMilli()) {
		return errors.New("attestation timestamp is older than unbonding period")
	}

	queryId, err := hex.DecodeString(request.QueryId)
	if err != nil {
		return err
	}
	checkpoint, err := hex.DecodeString(request.ValsetCheckpoint)
	if err != nil {
		return err
	}
	snapshotBytes, err := k.EncodeOracleAttestationData(
		queryId,
		request.Value,
		request.Timestamp,
		request.AggregatePower,
		request.PreviousTimestamp,
		request.NextTimestamp,
		checkpoint,
		request.AttestationTimestamp,
		request.LastConsensusTimestamp,
	)
	if err != nil {
		return err
	}

	// check whether snapshot exists. if it does, this attestation is not malicious
	// and we can return an error
	exists, err := k.SnapshotToAttestationsMap.Has(ctx, snapshotBytes)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("snapshot exists")
	}

	// check whether signature is associated with a valid operator address
	sigBytes, err := hex.DecodeString(request.Signature)
	if err != nil {
		return err
	}
	snapshotSha256 := sha256.Sum256(snapshotBytes)
	addrs, err := k.TryRecoverAddressWithBothIDs(sigBytes, snapshotSha256[:])
	if err != nil {
		return err
	}
	var operatorAddr types.OperatorAddress
	for _, addr := range addrs {
		evmAddrBytes := addr.Bytes()
		evmAddrString := common.Bytes2Hex(evmAddrBytes)
		k.Logger(ctx).Info("evmAddrString", "evmAddrString", evmAddrString)
		exists, err := k.EVMToOperatorAddressMap.Has(ctx, evmAddrString)
		if err != nil {
			return err
		}
		if exists {
			operatorAddr, err = k.EVMToOperatorAddressMap.Get(ctx, evmAddrString)
			if err != nil {
				return err
			}
			break
		}
	}

	if operatorAddr.OperatorAddress == nil {
		return errors.New("operator address not found")
	}

	// check the rate limit
	err = k.CheckRateLimit(ctx, operatorAddr, request.AttestationTimestamp)
	if err != nil {
		return err
	}

	// slash the validator
	slashFactor := math.LegacyNewDec(1).Quo(math.LegacyNewDec(100)) // update this to be a parameter in the bridge module
	err = k.SlashValidator(ctx, operatorAddr, slashFactor)
	if err != nil {
		return err
	}

	// set the attestation evidence submitted to true

	err = k.AttestationEvidenceSubmitted.Set(ctx, collections.Join(operatorAddr.OperatorAddress, request.AttestationTimestamp), types.BoolSubmitted{
		Submitted: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// SlashValidator slashes a validator for malicious attestation evidence.
func (k Keeper) SlashValidator(ctx context.Context, operatorAddr types.OperatorAddress, slashFactor math.LegacyDec) error {
	k.Logger(ctx).Info("slashValidator", "operatorAddr", operatorAddr.String())
	// get the validator address
	validatorAddr := sdk.ValAddress(operatorAddr.OperatorAddress)

	// get the validator
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return err
	}

	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return err
	}

	consAddrString, err := sdk.Bech32ifyAddressBytes(layerconfig.Bech32PrefixConsAddr, consAddr)
	if err != nil {
		return err
	}

	power := validator.ConsensusPower(validator.Tokens)
	k.Logger(ctx).Info("power", "power", power)
	k.Logger(ctx).Info("tokens", "tokens", validator.Tokens)

	adjustedPower := validator.GetConsensusPower(layertypes.PowerReduction)
	k.Logger(ctx).Info("adjustedPower", "adjustedPower", adjustedPower)
	slashAmount, err := k.stakingKeeper.SlashWithInfractionReason(ctx, consAddr, 1, adjustedPower, slashFactor, stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
	if err != nil {
		return err
	}
	k.Logger(ctx).Info("slashing validator", "consAddr", consAddrString, "slashAmount", slashAmount)
	// jail the validator -- uncomment this
	// err = k.stakingKeeper.Jail(ctx, consAddr)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// CheckRateLimit checks whether attestation evidence has been submitted for a given operator address with an attestation timestamp that is within the rate limit
func (k Keeper) CheckRateLimit(ctx context.Context, operatorAddr types.OperatorAddress, attestationTimestamp uint64) error {
	// get the rate limit. this should be updated to be a module parameter
	rateLimit := uint64(300000) // 5 minutes

	// first, check whether there is already a submission with this exact timestamp
	submitted, err := k.AttestationEvidenceSubmitted.Has(ctx, collections.Join(operatorAddr.OperatorAddress, attestationTimestamp))
	if err != nil {
		return err
	}
	if submitted {
		return errors.New("attestation evidence already submitted with this timestamp")
	}

	// check for evidence submitted before the attestation timestamp
	submitted, timestampBefore, _ := k.GetAttestationEvidenceSubmittedBefore(ctx, operatorAddr, attestationTimestamp)
	if submitted && attestationTimestamp >= timestampBefore {
		// check if the timestamp is within the rate limit
		if attestationTimestamp-timestampBefore < rateLimit {
			return errors.New("attestation evidence already submitted within rate limit")
		}
	}

	// check for evidence submitted after the attestation timestamp
	submitted, timestampAfter, _ := k.GetAttestationEvidenceSubmittedAfter(ctx, operatorAddr, attestationTimestamp)
	if submitted && timestampAfter >= attestationTimestamp {
		// check if the timestamp is within the rate limit
		if timestampAfter-attestationTimestamp < rateLimit {
			return errors.New("attestation evidence already submitted within rate limit")
		}
	}

	return nil
}

// GetAttestationEvidenceSubmittedBefore returns the timestamp of any attestation evidence submitted for a given operator address before a given timestamp
func (k Keeper) GetAttestationEvidenceSubmittedBefore(ctx context.Context, operatorAddress types.OperatorAddress, timestamp uint64) (submitted bool, timestampBefore uint64, err error) {
	// create a range that ends just before this timestamp
	rng := collections.NewPrefixedPairRange[[]byte, uint64](operatorAddress.OperatorAddress).EndExclusive(timestamp).Descending()

	var mostRecent bool
	var mostRecentTimestamp uint64

	// walk through the submissions in descending order to find the most recent one before the timestamp
	err = k.AttestationEvidenceSubmitted.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.BoolSubmitted) (stop bool, err error) {
		mostRecent = value.Submitted
		mostRecentTimestamp = key.K2()
		return true, nil // stop after the first (most recent) match
	})
	if err != nil {
		return false, 0, err
	}

	if !mostRecent {
		return false, 0, fmt.Errorf("no evidence submitted before timestamp %v for operator address %s", timestamp, operatorAddress.String())
	}

	return mostRecent, mostRecentTimestamp, nil
}

// GetAttestationEvidenceSubmittedAfter returns the timestamp of any attestation evidence submitted for a given operator address after a given timestamp
func (k Keeper) GetAttestationEvidenceSubmittedAfter(ctx context.Context, operatorAddr types.OperatorAddress, timestamp uint64) (bool, uint64, error) {
	rng := collections.NewPrefixedPairRange[[]byte, uint64](operatorAddr.OperatorAddress).StartExclusive(timestamp)

	var oldest bool
	var oldestTimestamp uint64

	err := k.AttestationEvidenceSubmitted.Walk(ctx, rng, func(key collections.Pair[[]byte, uint64], value types.BoolSubmitted) (stop bool, err error) {
		oldest = value.Submitted
		oldestTimestamp = key.K2()
		return true, nil
	})
	if err != nil {
		return false, 0, err
	}

	if !oldest {
		return false, 0, fmt.Errorf("no evidence submitted after timestamp %v for operator address %s", timestamp, operatorAddr.String())
	}

	return oldest, oldestTimestamp, nil
}
