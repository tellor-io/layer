package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	registryKeeper "github.com/tellor-io/layer/x/registry/keeper"
	registryTypes "github.com/tellor-io/layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.IsSenderStaked(ctx, msg.Creator) {
		return nil, status.Error(codes.Unauthenticated, "sender is not staked")
	}
	// check if query id is valid
	if !registryKeeper.IsQueryIdValid(msg.Qid) {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid query id: %s", msg.Qid))
	}
	// decode query id hex string to bytes
	qIdBytes, err := hex.DecodeString(msg.Qid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query ID string: %v", err))
	}
	// get query data from registry by query id
	queryData, err := k.registryKeeper.QueryData(ctx, msg.Qid)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("query data not found: %v", err))
	}
	// decode query data hex to get query type
	decodedQueryType, err := decodeQueryType(queryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query type: %v", err))
	}
	queryType := decodedQueryType[0].(string)
	// get data spec from registry by query type to validate value
	dataSpecBytes := k.registryKeeper.Spec(ctx, queryType)
	if dataSpecBytes == nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("data spec not found for query type: %s", queryType))
	}

	var dataSpec registryTypes.DataSpec
	k.cdc.Unmarshal(dataSpecBytes, &dataSpec)
	decodedSpec := &dataSpec
	valueType := decodedSpec.ValueType
	valueBytes, err := hex.DecodeString(msg.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode value string: %v", err))
	}
	// decode value using value type from data spec
	value, err := decodeValue(valueBytes, valueType)
	ctx.Logger().Info(fmt.Sprintf("value: %v", value[0]))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode value: %v", err))
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
	report := &types.MicroReport{
		Reporter:  msg.Creator,
		Qid:       msg.Qid,
		Value:     msg.Value,
		Timestamp: uint64(ctx.BlockTime().Unix()),
	}
	var reportsList types.Reports
	if err := k.cdc.Unmarshal(store.Get(qIdBytes), &reportsList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}
	reportsList.Reports = append(reportsList.Reports, report)
	store.Set(qIdBytes, k.cdc.MustMarshal(&reportsList))
	return &types.MsgSubmitValueResponse{}, nil
}

func decodeQueryType(data []byte) ([]interface{}, error) {
	// Create an ABI arguments object based on the types
	strArg, err := abi.NewType("string", "string", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
	}
	bytesArg, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
	}
	args := abi.Arguments{
		abi.Argument{Type: strArg},
		abi.Argument{Type: bytesArg},
	}
	result, err := args.UnpackValues(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack query type: %v", err)
	}
	return result, nil
}

func decodeValue(data []byte, dataType string) ([]interface{}, error) {
	argType, err := abi.NewType(dataType, dataType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when decoding value: %v", err)
	}
	arg := abi.Argument{
		Type: argType,
	}
	args := abi.Arguments{arg}
	var result []interface{}
	result, err = args.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack value: %v", err)
	}
	return result, nil
}

func (k Keeper) IsSenderStaked(ctx sdk.Context, sender string) bool {
	accAddr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return false
	}
	delegations := k.stakingKeeper.GetAllDelegatorDelegations(ctx, accAddr)
	var totalStakedTokens sdk.Dec
	for _, delegation := range delegations {
		totalStakedTokens = totalStakedTokens.Add(delegation.GetShares())
	}
	return totalStakedTokens.GT(sdk.ZeroDec())
}

// cleanup reports list
// func (k Keeper) CleanupReports(ctx sdk.Context, qid string) {
// 	qIdBytes, err := hex.DecodeString(qid)
// 	if err != nil {
// 		return
// 	}
// 	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
// 	var reportsList types.Reports
// 	if err := k.cdc.Unmarshal(store.Get(qIdBytes), &reportsList); err != nil {
// 		return
// 	}
// 	var newReportsList types.Reports
// 	// current time variable
// 	var currentTime = ctx.BlockTime().Unix()
// 	// if report.timestamp + 7days is less than current time, then delete the report
// 	for _, report := range reportsList.Reports {
// 		if report.Timestamp+604800 < uint64(currentTime) {
// 			continue
// 		}
// 		newReportsList.Reports = append(newReportsList.Reports, report)
// 	}

// 	store.Set(qIdBytes, k.cdc.MustMarshal(&newReportsList))
// }
