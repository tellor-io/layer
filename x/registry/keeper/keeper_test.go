package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestKeeper(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)

	unwrappedCtx := types.UnwrapSDKContext(ctx)

	// Logger
	logger := k.Logger(unwrappedCtx)
	loggerExpected := unwrappedCtx.Logger().With("module", fmt.Sprintf("x/%s", "registry"))
	require.Equal(t, logger, loggerExpected, "logger does not match")
}
