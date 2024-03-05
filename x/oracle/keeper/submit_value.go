package keeper

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
	regTypes "github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) setValue(ctx sdk.Context, reporter sdk.AccAddress, val string, queryData []byte, power, block int64) error {
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
	queryId := HashQueryData(queryData)
	report := types.MicroReport{
		Reporter:        reporter.String(),
		Power:           power,
		QueryType:       queryType,
		QueryId:         hex.EncodeToString(queryId),
		Value:           val,
		AggregateMethod: dataSpec.AggregationMethod,
		BlockNumber:     block,
		Timestamp:       ctx.BlockTime(),
	}

	return k.Reports.Set(ctx, collections.Join3(queryId, reporter.Bytes(), ctx.BlockHeight()), report)
}

func (k Keeper) VerifyCommit(ctx sdk.Context, reporter string, value, salt, hash string) bool {
	// calculate commitment
	calculatedCommit := utils.CalculateCommitment(value, salt)
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
