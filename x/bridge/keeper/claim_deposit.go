package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tellor-io/layer/lib/metrics"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) ClaimDeposit(ctx context.Context, depositId, timestamp uint64) error {
	cosmosCtx := sdk.UnwrapSDKContext(ctx)
	queryId, err := k.GetDepositQueryId(depositId)
	if err != nil {
		return err
	}
	aggregate, err := k.oracleKeeper.GetAggregateByTimestamp(ctx, queryId, timestamp)
	if err != nil {
		return err
	}
	if aggregate.Flagged {
		return types.ErrAggregateFlagged
	}
	depositClaimedStatus, err := k.DepositIdClaimedMap.Get(ctx, depositId)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	} else if depositClaimedStatus.Claimed {
		return types.ErrDepositAlreadyClaimed
	}
	// get power threshold at report time
	valsetTimestampBefore, err := k.GetValidatorSetTimestampBefore(ctx, timestamp)
	if err != nil {
		return err
	}
	valsetCheckpointParams, err := k.GetValidatorCheckpointParamsFromStorage(ctx, valsetTimestampBefore)
	if err != nil {
		return err
	}
	powerThreshold := valsetCheckpointParams.PowerThreshold
	if aggregate.AggregatePower < powerThreshold {
		return types.ErrInsufficientReporterPower
	}
	// ensure can't claim deposit until report is old enough
	if cosmosCtx.BlockTime().Sub(time.UnixMilli(int64(timestamp))) < 12*time.Hour {
		return types.ErrReportTooYoung
	}

	recipient, amount, _, err := k.DecodeDepositReportValue(ctx, aggregate.AggregateValue)
	if err != nil {
		k.Logger(ctx).Error("claimDeposit", "error", fmt.Errorf("failed to decode deposit report value, err: %w", err))
		return fmt.Errorf("%s: %w", types.ErrInvalidDepositReportValue.Error(), err)
	}

	newClaimedStatus := types.DepositClaimed{Claimed: true}
	err = k.DepositIdClaimedMap.Set(ctx, depositId, newClaimedStatus)
	if err != nil {
		k.Logger(ctx).Error("Failed to set deposit claimed status", "depositId", depositId, "err", err)
		return err
	}

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, amount); err != nil {
		k.Logger(ctx).Error("claimDeposit", "error", fmt.Errorf("failed to mint coins, err: %w", err))
		return err
	}

	claimAmount := amount
	// if tip.IsAllPositive() {
	// 	claimAmount = amount.Sub(tip...)
	// 	err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, msgSender, tip)
	// 	if err != nil {
	// 		k.Logger(ctx).Error("claimDeposit", "error", fmt.Errorf("failed to send coins, err: %w", err))
	// 		return err
	// 	}
	// }

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, claimAmount); err != nil {
		k.Logger(ctx).Error("claimDeposit", "error", fmt.Errorf("failed to send coins, err: %w", err))
		return err
	}

	telemetry.IncrCounterWithLabels([]string{"claimed_deposit_tracker"}, float32(amount.AmountOf("loya").Int64()), []metrics.Label{{Name: "chain_id", Value: cosmosCtx.ChainID()}})
	return nil
}

// replicate solidity encoding,  keccak256(abi.encode(string "TRBBridge", abi.encode(bool true, uint256 depositId)))
func (k Keeper) GetDepositQueryId(depositId uint64) ([]byte, error) {
	queryTypeString := "TRBBridge"
	toLayerBool := true
	depositIdUint64 := new(big.Int).SetUint64(depositId)

	// prepare encoding
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	BoolType, err := abi.NewType("bool", "", nil)
	if err != nil {
		return nil, err
	}
	BytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	// encode query data arguments first
	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}
	queryDataArgsEncoded, err := queryDataArgs.Pack(toLayerBool, depositIdUint64)
	if err != nil {
		return nil, err
	}

	// encode query data
	finalArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataEncoded, err := finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	if err != nil {
		return nil, err
	}

	// generate query id
	queryId := crypto.Keccak256(queryDataEncoded)
	return queryId, nil
}

// replicate solidity decoding, abi.decode(reportValue, (address ethSender, string layerRecipient, uint256 amount, uint256 tip))
func (k Keeper) DecodeDepositReportValue(ctx context.Context, reportValue string) (recipient sdk.AccAddress, amount, tip sdk.Coins, err error) {
	// prepare decoding
	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, sdk.Coins{}, sdk.Coins{}, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, sdk.Coins{}, sdk.Coins{}, err
	}
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, sdk.Coins{}, sdk.Coins{}, err
	}
	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
		{Type: Uint256Type},
	}
	// decode report value
	reportValueBytes, err := hex.DecodeString(reportValue)
	if err != nil {
		k.Logger(ctx).Error("DecodeDepositReportValue", "error", fmt.Errorf("failed to decode report value, err: %w", err))
		return nil, sdk.Coins{}, sdk.Coins{}, err
	}
	reportValueDecoded, err := reportValueArgs.Unpack(reportValueBytes)
	if err != nil {
		k.Logger(ctx).Error("DecodeDepositReportValue", "error", fmt.Errorf("failed to decode report value, err: %w", err))
		return nil, sdk.Coins{}, sdk.Coins{}, err
	}
	recipientString := reportValueDecoded[1].(string)
	amountBigInt := reportValueDecoded[2].(*big.Int)
	tipBigInt := reportValueDecoded[3].(*big.Int)
	// convert layer recipient to cosmos address
	layerRecipientAddress, err := sdk.AccAddressFromBech32(recipientString)
	if err != nil {
		k.Logger(ctx).Error("DecodeDepositReportValue", "error", fmt.Errorf("failed to convert layer recipient to cosmos address, err: %w", err))
		// use team address as recipient if conversion fails
		layerRecipientAddress, err = k.disputeKeeper.GetTeamAddress(ctx)
		if err != nil {
			k.Logger(ctx).Error("DecodeDepositReportValue", "error", fmt.Errorf("failed to get team address, err: %w", err))
			return nil, sdk.Coins{}, sdk.Coins{}, err
		}
	}
	amountDecimalConverted := amountBigInt.Div(amountBigInt, big.NewInt(1e12))
	tipDecimalConverted := tipBigInt.Div(tipBigInt, big.NewInt(1e12))
	amountCoin := sdk.NewInt64Coin(layer.BondDenom, amountDecimalConverted.Int64())
	amountCoins := sdk.NewCoins(amountCoin)
	tipCoin := sdk.NewInt64Coin(layer.BondDenom, tipDecimalConverted.Int64())
	tipCoins := sdk.NewCoins(tipCoin)

	return layerRecipientAddress, amountCoins, tipCoins, nil
}

func (k Keeper) GetDepositStatus(ctx context.Context, depositId uint64) (bool, error) {
	claimed, err := k.DepositIdClaimedMap.Get(ctx, depositId)
	if err != nil {
		return false, err
	}
	return claimed.Claimed, nil
}
