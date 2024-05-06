package keeper_test

import (
	"fmt"
	"testing"

	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestNewKeeper(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
}

func TestGetAuthority(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	queryType := "testQueryType"
	spec := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
	}

	// Register spec
	registerSpecInput := &types.MsgRegisterSpec{
		Registrar: "creator1",
		QueryType: queryType,
		Spec:      spec,
	}
	_, err := ms.RegisterSpec(ctx, registerSpecInput)
	require.NoError(t, err)
	_a := k.GetAuthority()
	require.Equal(t, authority, _a)
}

func TestLogger(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	unwrappedCtx := sdkTypes.UnwrapSDKContext(ctx)

	// Logger
	logger := k.Logger(unwrappedCtx)
	loggerExpected := unwrappedCtx.Logger().With("module", fmt.Sprintf("x/%s", "registry"))
	require.Equal(t, logger, loggerExpected, "logger does not match")
}

func TestSetHooks(t *testing.T) {
}

func TestHooks(t *testing.T) {
}
