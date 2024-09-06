package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	regTypes "github.com/tellor-io/layer/x/registry/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetValue(ctx context.Context, reporter sdk.AccAddress, query types.QueryMeta, val string, queryData []byte, power int64, incycle bool) error {
	// decode query data hex to get query type, returns interface array
	queryType, _, err := regTypes.DecodeQueryType(queryData)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query type: %v", err))
	}
	dataSpec, err := k.GetDataSpec(ctx, queryType)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get value type: %v", err))
	}
	// decode value using value type from data spec and check if decodes successfully
	// value is not used, only used to check if it decodes successfully
	if err := dataSpec.ValidateValue(val); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to validate value: %v", err))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	queryId := utils.QueryIDFromData(queryData)
	report := types.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryType:       queryType,
		QueryId:         queryId,
		Value:           val,
		AggregateMethod: dataSpec.AggregationMethod,
		Timestamp:       sdkCtx.BlockTime(),
		Cyclelist:       incycle,
		BlockNumber:     sdkCtx.BlockHeight(),
	}

	query.HasRevealedReports = true
	err = k.Query.Set(ctx, collections.Join(queryId, query.Id), query)
	if err != nil {
		return err
	}
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			"new_report",
			sdk.NewAttribute("reporter", reporter.String()),
			sdk.NewAttribute("reporter_power", fmt.Sprintf("%d", power)),
			sdk.NewAttribute("query_type", queryType),
			sdk.NewAttribute("query_id", hex.EncodeToString(queryId)),
			sdk.NewAttribute("value", val),
			sdk.NewAttribute("cyclelist", fmt.Sprintf("%t", incycle)),
			sdk.NewAttribute("aggregate_method", dataSpec.AggregationMethod),
			sdk.NewAttribute("query_data", hex.EncodeToString(queryData)),
		),
	})
	return k.Reports.Set(ctx, collections.Join3(queryId, reporter.Bytes(), query.Id), report)
}

func (k Keeper) VerifyCommit(ctx context.Context, reporter, value, salt, hash string) bool {
	// calculate commitment
	calculatedCommit := oracleutils.CalculateCommitment(value, salt)
	// compare calculated commitment with the one stored
	return calculatedCommit == hash
}

func (k Keeper) GetDataSpec(ctx context.Context, queryType string) (regTypes.DataSpec, error) {
	// get data spec from registry by query type to validate value
	dataSpec, err := k.registryKeeper.GetSpec(ctx, queryType)
	if err != nil {
		return regTypes.DataSpec{}, err
	}
	return dataSpec, nil
}
