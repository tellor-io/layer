package keeper

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/oracle/utils"
	regTypes "github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) setValue(ctx sdk.Context, reporter sdk.AccAddress, val string, queryData []byte, power, block int64) error {
	// decode query data hex to get query type, returns interface array
	queryType, err := decodeQueryType(queryData)
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

func (k Keeper) IsReporterStaked(ctx sdk.Context, reporter sdk.ValAddress) (int64, bool) {

	validator, err := k.stakingKeeper.Validator(ctx, reporter)
	if err != nil {
		// TODO: return errors
		panic(err)
	}
	if validator == nil {
		return 0, false
	}
	// check if validator is active
	if validator.IsJailed() || validator.IsUnbonding() || validator.IsUnbonded() {
		return 0, false
	}
	// get voting power
	votingPower := validator.GetConsensusPower(sdk.DefaultPowerReduction)

	return votingPower, validator.IsBonded()
}

// tODO: double check this // no longer needed ?
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

func (k Keeper) VerifyCommit(ctx sdk.Context, reporter string, value, salt, hash string) bool {
	// calculate commitment
	calculatedCommit := utils.CalculateCommitment(value, salt)
	// compare calculated commitment with the one stored
	return calculatedCommit == hash
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

func (k Keeper) GetDataSpec(ctx sdk.Context, queryType string) (regTypes.DataSpec, error) {
	// get data spec from registry by query type to validate value
	dataSpec, err := k.registryKeeper.GetSpec(ctx, queryType)
	if err != nil {
		return regTypes.DataSpec{}, err
	}
	return dataSpec, nil
}
