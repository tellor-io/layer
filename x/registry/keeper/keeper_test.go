package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/registry/types"

	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	testQueryType = "testQuerytype"
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
	registrar := sample.AccAddress()
	spec := types.DataSpec{
		DocumentHash:      "testHash",
		ResponseValueType: "uint256",
		AggregationMethod: "weighted-median",
		QueryType:         testQueryType,
		Registrar:         registrar,
		AbiComponents: []*types.ABIComponent{
			{
				Name:      "field",
				FieldType: "uint256",
			},
		},
	}

	// Register spec
	registerSpecInput := &types.MsgRegisterSpec{
		Registrar: registrar,
		QueryType: testQueryType,
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

func TestSetHooksAndHooks(t *testing.T) {
	_k2, _, _, _, _, _ := keepertest.OracleKeeper(t)
	_, _, k := setupMsgServer(t)
	k.SetHooks(_k2.Hooks())
	_h := k.Hooks()
	require.NotNil(t, _h)
}
