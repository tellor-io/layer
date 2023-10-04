package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	registryKeeper "github.com/tellor-io/layer/x/registry/keeper"
	registryTypes "github.com/tellor-io/layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) SubmitValue(goCtx context.Context, msg *types.MsgSubmitValue) (*types.MsgSubmitValueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// convert reporter address from bech32 to sdk.AccAddress
	reporter, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode reporter address: %v", err))
	}
	// check if sender is bonded
	if !k.IsReporterStaked(ctx, reporter) {
		return nil, status.Error(codes.Unauthenticated, "sender is not staked")
	}
	// check if querydata has prefix 0x
	if registryKeeper.Has0xPrefix(msg.QueryData) {
		msg.QueryData = msg.QueryData[2:]
	}
	// decode query data hex string to bytes
	queryData, err := hex.DecodeString(msg.QueryData)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query data string: %v", err))
	}
	// get commit from store
	commitValue, err := k.getSignature(ctx, msg.Creator, HashQueryData(queryData))
	if err != nil {
		return nil, err
	}
	// check if value is being revealed in the one block after commit
	if ctx.BlockHeight()-1 != commitValue.Block {
		return nil, status.Error(codes.InvalidArgument, "missed block height to reveal")
	}
	// if commitValue.Block < ctx.BlockHeight()-5 || commitValue.Block > ctx.BlockHeight() {
	// 	return nil, status.Error(codes.InvalidArgument, "missed block height window to reveal")
	// }
	// verify value signature
	if !k.verifySignature(ctx, reporter, msg.Value, commitValue.Report.Signature) {
		return nil, status.Error(codes.InvalidArgument, "unable to verify signature")
	}
	// set value
	if err := k.setValue(ctx, msg.Creator, msg.Value, queryData); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to set value: %v", err))
	}
	// emit event
	err = ctx.EventManager().EmitTypedEvent(msg)
	if err != nil {
		return nil, err
	}
	return &types.MsgSubmitValueResponse{}, nil
}

func (k Keeper) setValue(ctx sdk.Context, reporter, val string, queryData []byte) error {
	queryId := HashQueryData(queryData)
	queryType, err := decodeQueryType(queryData)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode query type: %v", err))
	}
	valueType, err := getValueType(k.registryKeeper, k.cdc, ctx, queryType)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to get value type: %v", err))
	}
	// decode value using value type from data spec and check if decodes successfully
	// value is not used, only used to check if it decodes successfully
	value, err := decodeValue(val, valueType)
	ctx.Logger().Info(fmt.Sprintf("value: %v", value[0]))
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode value: %v", err))
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
	report := &types.MicroReport{
		Reporter:  reporter,
		Qid:       hex.EncodeToString(queryId),
		Value:     val,
		Timestamp: uint64(ctx.BlockTime().Unix()),
	}
	var reportsList types.Reports
	if err := k.cdc.Unmarshal(store.Get(queryId), &reportsList); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to unmarshal reports: %v", err))
	}
	reportsList.MicroReports = append(reportsList.MicroReports, report)
	store.Set(queryId, k.cdc.MustMarshal(&reportsList))
	return nil
}
func getValueType(registry types.RegistryKeeper, cdc codec.BinaryCodec, ctx sdk.Context, queryType string) (string, error) {
	// get data spec from registry by query type to validate value
	dataSpecBytes := registry.Spec(ctx, queryType)
	if dataSpecBytes == nil {
		return "", status.Error(codes.NotFound, fmt.Sprintf("data spec not found for query type: %s", queryType))
	}
	var dataSpec registryTypes.DataSpec
	cdc.Unmarshal(dataSpecBytes, &dataSpec)

	return dataSpec.ValueType, nil
}

func (k Keeper) IsReporterStaked(ctx sdk.Context, reporter sdk.AccAddress) bool {
	delegations := k.stakingKeeper.GetAllDelegatorDelegations(ctx, reporter)

	var totalStakedTokens sdk.Dec = sdk.ZeroDec()
	for _, delegation := range delegations {
		totalStakedTokens = totalStakedTokens.Add(delegation.Shares)
	}
	return totalStakedTokens.GT(sdk.ZeroDec())
}

func (k Keeper) verifySignature(ctx sdk.Context, reporter sdk.AccAddress, value, signature string) bool {
	reporterAccount := k.accountKeeper.GetAccount(ctx, reporter)
	pubKey := reporterAccount.GetPubKey()
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	// decode value from hex string
	valBytes, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	// verify signature
	if !pubKey.VerifySignature(valBytes, sigBytes) {
		return false
	}
	return true
}

func decodeQueryType(data []byte) (string, error) {
	// Create an ABI arguments object based on the types
	strArg, err := abi.NewType("string", "string", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
	}
	bytesArg, err := abi.NewType("bytes", "bytes", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create new ABI type when decoding query type: %v", err)
	}
	args := abi.Arguments{
		abi.Argument{Type: strArg},
		abi.Argument{Type: bytesArg},
	}
	result, err := args.UnpackValues(data)
	if err != nil {
		return "", fmt.Errorf("failed to unpack query type: %v", err)
	}
	return result[0].(string), nil
}

func decodeValue(value, dataType string) ([]interface{}, error) {
	argType, err := abi.NewType(dataType, dataType, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ABI type when decoding value: %v", err)
	}
	arg := abi.Argument{
		Type: argType,
	}
	args := abi.Arguments{arg}
	valueBytes, err := hex.DecodeString(value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode value string: %v", err))
	}
	var result []interface{}
	result, err = args.Unpack(valueBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack value: %v", err)
	}
	return result, nil
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
