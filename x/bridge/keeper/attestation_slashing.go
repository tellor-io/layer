package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	layerconfig "github.com/tellor-io/layer/app/config"
	layertypes "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

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

	// slash the validator
	err = k.slashValidator(ctx, operatorAddr)
	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) slashValidator(ctx context.Context, operatorAddr types.OperatorAddress) error {
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
	slashFactor := math.LegacyNewDec(1).Quo(math.LegacyNewDec(100))
	slashAmount, err := k.stakingKeeper.SlashWithInfractionReason(ctx, consAddr, 1, adjustedPower, slashFactor, stakingtypes.Infraction_INFRACTION_UNSPECIFIED)
	if err != nil {
		return err
	}
	k.Logger(ctx).Info("slashing validator", "consAddr", consAddrString, "slashAmount", slashAmount)
	return nil
}
