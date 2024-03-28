package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	regTypes "github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) setValue(ctx sdk.Context, reporter sdk.AccAddress, query types.QueryMeta, val string, queryData []byte, power int64, incycle bool) error {
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
	queryId := utils.QueryIDFromData(queryData)
	report := types.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryType:       queryType,
		QueryId:         queryId,
		Value:           val,
		AggregateMethod: dataSpec.AggregationMethod,
		Timestamp:       ctx.BlockTime(),
		Cyclelist:       incycle,
		BlockNumber:     ctx.BlockHeight(),
	}

	query.HasRevealedReports = true
	err = k.Query.Set(ctx, queryId, query)
	if err != nil {
		return err
	}

	return k.Reports.Set(ctx, collections.Join3(queryId, reporter.Bytes(), query.Id), report)
}

func (k Keeper) VerifyCommit(ctx sdk.Context, reporter string, value, salt, hash string) bool {
	// calculate commitment
	calculatedCommit := oracleutils.CalculateCommitment(value, salt)
	// compare calculated commitment with the one stored
	return calculatedCommit == hash
}

func (k Keeper) GetDataSpec(ctx sdk.Context, queryType string) (regTypes.DataSpec, error) {
	// get data spec from registry by query type to validate value
	dataSpec, err := k.registryKeeper.GetSpec(ctx, queryType)
	if err != nil {
		return regTypes.DataSpec{}, err
	}
	return dataSpec, nil
}
