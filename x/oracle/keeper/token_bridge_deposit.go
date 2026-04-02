package keeper

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/x/oracle/types"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	TRBBridgeQueryType   = "TRBBridge"
	TRBBridgeV2QueryType = "TRBBridgeV2"
)

// Generates a new QueryMeta for a TRBBridgeV2QueryType
func (k Keeper) TokenBridgeDepositQuery(ctx context.Context, queryData []byte) (types.QueryMeta, error) {
	// decode query data partial
	nextId, err := k.QuerySequencer.Next(ctx)
	if err != nil {
		return types.QueryMeta{}, err
	}
	// get block timestamp
	query := types.QueryMeta{
		Id:                      nextId,
		RegistrySpecBlockWindow: 2000,
		Expiration:              uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()) + 2000,
		QueryType:               TRBBridgeV2QueryType,
		QueryData:               queryData,
		Amount:                  math.NewInt(0),
		CycleList:               true,
	}

	return query, nil
}

func (k Keeper) HandleBridgeDepositDirectReveal(
	ctx context.Context,
	query types.QueryMeta,
	querydata []byte,
	reporterAcc sdk.AccAddress,
	value string,
	voterPower uint64,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockHeight := sdkCtx.BlockHeight()

	if query.Amount.IsZero() && query.Expiration <= uint64(blockHeight) {
		nextId, err := k.QuerySequencer.Next(ctx)
		if err != nil {
			return err
		}
		query.Id = nextId
		query.Expiration = uint64(blockHeight) + query.RegistrySpecBlockWindow
	}
	if query.Amount.GT(math.ZeroInt()) && query.Expiration <= uint64(blockHeight) {
		query.Expiration = uint64(blockHeight) + query.RegistrySpecBlockWindow
	}

	if query.Expiration < uint64(blockHeight) {
		return types.ErrSubmissionWindowExpired.Wrapf("query for bridge deposit is expired")
	}

	if err := validateBridgeDepositAmount(value); err != nil {
		return err
	}

	return k.SetValue(ctx, reporterAcc, query, value, querydata, voterPower, true)
}

// this is a simpler version of DecodeDepositReportValue in x/bridge/keeper
// rewriting to avoid circular dependencies
func validateBridgeDepositAmount(value string) error {
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return types.ErrInvalidValue.Wrap("failed to create address type")
	}
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return types.ErrInvalidValue.Wrap("failed to create string type")
	}
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return types.ErrInvalidValue.Wrap("failed to create uint256 type")
	}

	args := abi.Arguments{
		{Type: addressType},
		{Type: stringType},
		{Type: uint256Type},
		{Type: uint256Type},
	}
	valueBytes, err := hex.DecodeString(registrytypes.Remove0xPrefix(value))
	if err != nil {
		return types.ErrInvalidValue.Wrap("failed to decode bridge deposit value hex")
	}
	decoded, err := args.Unpack(valueBytes)
	if err != nil {
		return types.ErrInvalidValue.Wrap("failed to decode bridge deposit value")
	}
	amount := decoded[2].(*big.Int)
	if amount.Sign() <= 0 {
		return types.ErrInvalidValue.Wrap("bridge deposit amount cannot be zero")
	}
	return nil
}
