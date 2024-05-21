package keeper

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

func (k Keeper) ClaimDeposit(ctx context.Context, depositId uint64, reportIndex uint64) error {
	cosmosCtx := sdk.UnwrapSDKContext(ctx)
	queryId, err := k.GetDepositQueryId(depositId)
	if err != nil {
		return err
	}
	aggregate, aggregateTimestamp, err := k.oracleKeeper.GetAggregateByIndex(ctx, queryId, reportIndex)
	if err != nil {
		return err
	}
	if aggregate == nil {
		return types.ErrNoAggregate
	}
	if aggregate.Flagged {
		return types.ErrAggregateFlagged
	}
	depositClaimedStatus, err := k.DepositIdClaimedMap.Get(ctx, depositId)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return err
	} else {
		if depositClaimedStatus.Claimed {
			return types.ErrDepositAlreadyClaimed
		}
	}
	// get total bonded tokens
	totalBondedTokens, err := k.reporterKeeper.TotalReporterPower(ctx)
	if err != nil {
		return err
	}
	powerThreshold := int64(math.Round(float64(totalBondedTokens.Int64()) * 2 / 3))
	if aggregate.ReporterPower < powerThreshold {
		return types.ErrInsufficientReporterPower
	}
	// ensure can't claim deposit until report is old enough
	if cosmosCtx.BlockTime().Sub(aggregateTimestamp) < 12*time.Hour {
		return types.ErrReportTooYoung
	}

	recipient, amount, err := k.DecodeDepositReportValue(ctx, aggregate.AggregateValue)
	if err != nil {
		k.Logger(ctx).Error("@claimDeposit", "error", fmt.Errorf("failed to decode deposit report value, err: %w", err))
		return fmt.Errorf("%w: %v", types.ErrInvalidDepositReportValue, err)
	}

	newClaimedStatus := types.DepositClaimed{Claimed: true}
	err = k.DepositIdClaimedMap.Set(ctx, depositId, newClaimedStatus)
	if err != nil {
		k.Logger(ctx).Error("Failed to set deposit claimed status", "depositId", depositId, "err", err)
		return err
	}

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, amount); err != nil {
		k.Logger(ctx).Error("@claimDeposit", "error", fmt.Errorf("failed to mint coins, err: %w", err))
		return err
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, amount); err != nil {
		k.Logger(ctx).Error("@claimDeposit", "error", fmt.Errorf("failed to send coins, err: %w", err))
		return err
	}

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

// replicate solidity decoding, abi.decode(reportValue, (address ethSender, string layerRecipient, uint256 amount))
func (k Keeper) DecodeDepositReportValue(ctx context.Context, reportValue string) (recipient sdk.AccAddress, amount sdk.Coins, err error) {

	// prepare decoding
	AddressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, sdk.Coins{}, err
	}
	Uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, sdk.Coins{}, err
	}
	StringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, sdk.Coins{}, err
	}

	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
	}

	// decode report value
	reportValueBytes, err := hex.DecodeString(reportValue)
	if err != nil {
		k.Logger(ctx).Error("@decodeDepositReportValue", "error", fmt.Errorf("failed to decode report value, err: %w", err))
		return nil, sdk.Coins{}, err
	}
	reportValueDecoded, err := reportValueArgs.Unpack(reportValueBytes)
	if err != nil {
		k.Logger(ctx).Error("@decodeDepositReportValue", "error", fmt.Errorf("failed to decode report value, err: %w", err))
		return nil, sdk.Coins{}, err
	}

	recipientString := reportValueDecoded[1].(string)
	amountBigInt := reportValueDecoded[2].(*big.Int)

	// convert layer recipient to cosmos address
	layerRecipientAddress, err := sdk.AccAddressFromBech32(recipientString)
	if err != nil {
		k.Logger(ctx).Error("@decodeDepositReportValue", "error", fmt.Errorf("failed to convert layer recipient to cosmos address, err: %w", err))
		return nil, sdk.Coins{}, err
	}

	amountDecimalConverted := amountBigInt.Div(amountBigInt, big.NewInt(1e12))

	amountCoin := sdk.NewInt64Coin(layer.BondDenom, amountDecimalConverted.Int64())
	amountCoins := sdk.NewCoins(amountCoin)

	return layerRecipientAddress, amountCoins, nil
}
