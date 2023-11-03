package keeper

import (
	"encoding/hex"
	"fmt"

	"github.com/tellor-io/layer/x/oracle/types"
	registryTypes "github.com/tellor-io/layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) GetSignature(ctx sdk.Context, reporter string, queryId []byte) (*types.CommitValue, error) {

	commitStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CommitReportKey))
	commit := commitStore.Get(append([]byte(reporter), queryId...))
	if commit == nil {
		return nil, status.Error(codes.NotFound, "no commits to reveal found")
	}
	var commitValue types.CommitValue
	k.cdc.Unmarshal(commit, &commitValue)
	return &commitValue, nil
}

func (k Keeper) setValueByReporter(ctx sdk.Context, reporter, val string, queryId []byte, power int64) error {
	queryIdHex := hex.EncodeToString(queryId)
	reporterStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReporterStoreKey))
	// reporter-query id pair
	reporterQueryIdKey := []byte(reporter + ":" + queryIdHex)
	// get reports list from store and unmarshal
	var reportsList types.Reports
	if err := k.cdc.Unmarshal(reporterStore.Get(reporterQueryIdKey), &reportsList); err != nil {
		return fmt.Errorf("failed to unmarshal reports: %v", err)
	}
	report := &types.MicroReport{
		Reporter:  reporter,
		Power:     power,
		QueryId:   queryIdHex,
		Value:     val,
		Timestamp: ctx.BlockTime(),
	}
	// append report to reports list
	reportsList.MicroReports = append(reportsList.MicroReports, report)
	reporterStore.Set(reporterQueryIdKey, k.cdc.MustMarshal(&reportsList))
	return nil
}

func (k Keeper) setValueByQueryId(ctx sdk.Context, reporter, val string, queryId []byte, power int64) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ReportsKey))
	report := &types.MicroReport{
		Reporter:  reporter,
		Power:     power,
		QueryId:   hex.EncodeToString(queryId),
		Value:     val,
		Timestamp: ctx.BlockTime(),
	}
	var reportsList types.Reports
	if err := k.cdc.Unmarshal(store.Get(queryId), &reportsList); err != nil {
		return fmt.Errorf("failed to unmarshal reports: %v", err)
	}
	reportsList.MicroReports = append(reportsList.MicroReports, report)
	store.Set(queryId, k.cdc.MustMarshal(&reportsList))
	return nil
}

func (k Keeper) setValue(ctx sdk.Context, reporter, val string, queryData []byte, power int64) error {
	queryId := HashQueryData(queryData)
	// decode query data hex to get query type, returns interface array
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
	// set value by reporter
	if err := k.setValueByReporter(ctx, reporter, val, queryId, power); err != nil {
		return err
	}
	// set value by query id
	if err := k.setValueByQueryId(ctx, reporter, val, queryId, power); err != nil {
		return err
	}
	return nil
}

func (k Keeper) IsReporterStaked(ctx sdk.Context, reporter sdk.ValAddress) (int64, bool) {

	validator := k.stakingKeeper.Validator(ctx, reporter)
	if validator == nil {
		return 0, false
	}
	// check if validator is active
	if validator.IsJailed() || validator.IsUnbonding() || validator.IsUnbonded() {
		return 0, false
	}
	// get voting power
	votingPower := validator.GetConsensusPower(validator.GetBondedTokens())

	return votingPower, validator.IsBonded()
}

func (k Keeper) VerifySignature(ctx sdk.Context, reporter string, value, signature string) bool {
	addr, err := sdk.AccAddressFromBech32(reporter)
	if err != nil {
		return false
	}
	reporterAccount := k.accountKeeper.GetAccount(ctx, addr)
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
