package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestReportersQuery(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	for i := 0; i < 10; i++ {
		err := k.Reporters.Set(ctx, sample.AccAddressBytes(), types.NewReporter(types.DefaultMinCommissionRate, types.DefaultMinTrb))
		require.NoError(t, err)
	}
	res, err := querier.Reporters(ctx, &types.QueryReportersRequest{})
	require.NoError(t, err)
	require.Len(t, res.Reporters, 10)
}

func TestSelectorReporterQuery(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	querier := keeper.NewQuerier(k)
	selector := sample.AccAddressBytes()
	reporterAddr := sample.AccAddressBytes()
	err := k.Selectors.Set(ctx, selector, types.NewSelection(reporterAddr, 1))
	require.NoError(t, err)
	res, err := querier.SelectorReporter(ctx, &types.QuerySelectorReporterRequest{SelectorAddress: selector.String()})
	require.NoError(t, err)
	require.Equal(t, reporterAddr.String(), res.Reporter)
}
