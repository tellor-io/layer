package keeper_test

import (
	"fmt"
	"testing"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/registry/keeper"
	//types "github.com/tellor-io/layer/x/registry/types"
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

	k.SetGenesisSpec(unwrappedCtx)
	k.SetGenesisQuery(unwrappedCtx)

	//GetGenesisSpec()
	genesisSpec := k.GetGenesisSpec(unwrappedCtx)
	genesisHash := genesisSpec.DocumentHash
	require.Equal(t, genesisHash, "", "hashes do no match")
	genesisType := genesisSpec.ValueType
	require.Equal(t, genesisType, "uint256", "types do no match")

	// GetGenesisQuery()
	trbQuery, btcQuery, ethQuery := k.GetGenesisQuery(unwrappedCtx)
	trbQueryData := keeper.SpotQueryData("trb", "usd")
	require.Equal(t, trbQuery, bytes.HexBytes(trbQueryData).String(), "trb query data doesnt match")
	btcQueryData := keeper.SpotQueryData("btc", "usd")
	require.Equal(t, btcQuery, bytes.HexBytes(btcQueryData).String(), "btc query data doesnt match")
	ethQueryData := keeper.SpotQueryData("eth", "usd")
	require.Equal(t, ethQuery, bytes.HexBytes(ethQueryData).String(), "eth query data doesnt match")

}
