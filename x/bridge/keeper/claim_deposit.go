package keeper

import (
	"context"
	"fmt"
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/bridge/types"
)

func (k Keeper) claimDeposit(ctx context.Context, depositId uint64, reportIndex uint64) error {
	cosmosCtx := sdk.UnwrapSDKContext(ctx)
	queryId, err := k.getDepositQueryId(depositId)
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
	if err != nil {
		return err
	}
	if depositClaimedStatus.Claimed {
		return types.ErrDepositAlreadyClaimed
	}
	// get total bonded tokens
	totalBondedTokens, err := k.stakingKeeper.TotalBondedTokens(ctx)
	if err != nil {
		return err
	}
	powerThreshold := totalBondedTokens.Int64() * 2 / 3
	if aggregate.ReporterPower < powerThreshold {
		return types.ErrInsufficientReporterPower
	}
	// ensure can't claim deposit until report is old enough
	if cosmosCtx.BlockTime().Sub(aggregateTimestamp) < 1*time.Minute {
		return types.ErrReportTooYoung
	}

	recipient, amount, err := k.decodeDepositReportValue(aggregate.AggregateValue)
	if err != nil {
		return fmt.Errorf("%w: %v", types.ErrInvalidDepositReportValue, err)
	}

	if err := k.DepositIdClaimedMap.Set(ctx, depositId, types.DepositClaimed{Claimed: true}); err != nil {
		return err
	}

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, amount); err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, amount); err != nil {
		return err
	}

	return nil
}

// type Aggregate struct {
//     QueryId              []byte               `protobuf:"bytes,1,opt,name=query_id,json=queryId,proto3" json:"query_id,omitempty"`
//     AggregateValue       string               `protobuf:"bytes,2,opt,name=aggregateValue,proto3" json:"aggregateValue,omitempty"`
//     AggregateReporter    string               `protobuf:"bytes,3,opt,name=aggregateReporter,proto3" json:"aggregateReporter,omitempty"`
//     ReporterPower        int64                `protobuf:"varint,4,opt,name=reporterPower,proto3" json:"reporterPower,omitempty"`
//     StandardDeviation    float64              `protobuf:"fixed64,5,opt,name=standardDeviation,proto3" json:"standardDeviation,omitempty"`
//     Reporters            []*AggregateReporter `protobuf:"bytes,6,rep,name=reporters,proto3" json:"reporters,omitempty"`
//     Flagged              bool                 `protobuf:"varint,7,opt,name=flagged,proto3" json:"flagged,omitempty"`
//     Nonce                uint64               `protobuf:"varint,8,opt,name=nonce,proto3" json:"nonce,omitempty"`
//     AggregateReportIndex int64                `protobuf:"varint,9,opt,name=aggregateReportIndex,proto3" json:"aggregateReportIndex,omitempty"`
//     Height               int64                `protobuf:"varint,10,opt,name=height,proto3" json:"height,omitempty"`
// }

func (k Keeper) getDepositQueryId(depositId uint64) ([]byte, error) {
	// replicate solidity encoding,  keccak256(abi.encode(string "TRBBridge", abi.encode(bool true, uint256 depositId)))

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

func (k Keeper) decodeDepositReportValue(reportValue string) (recipient sdk.AccAddress, amount sdk.Coins, err error) {
	// replicate solidity decoding, abi.decode(reportValue, (address ethSender, string layerRecipient, uint256 amount))

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
	reportValueDecoded, err := reportValueArgs.Unpack([]byte(reportValue))
	if err != nil {
		return nil, sdk.Coins{}, err
	}

	recipientString := reportValueDecoded[1].(string)
	amountBigInt := reportValueDecoded[2].(*big.Int)

	// convert layer recipient to cosmos address
	layerRecipientAddress, err := sdk.AccAddressFromBech32(recipientString)
	if err != nil {
		return nil, sdk.Coins{}, err
	}

	amountDecimalConverted := amountBigInt.Int64() / 1e12

	amountCoin := sdk.NewInt64Coin(layer.BondDenom, amountDecimalConverted)
	amountCoins := sdk.NewCoins(amountCoin)

	return layerRecipientAddress, amountCoins, nil
}
