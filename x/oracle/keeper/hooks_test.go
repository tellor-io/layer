package keeper_test

import (
	"time"

	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
)

func (s *KeeperTestSuite) TestHooks() {
	require := s.Require()
	k := s.oracleKeeper

	hooks := k.Hooks()
	require.NotNil(hooks)
}

func (s *KeeperTestSuite) TestAfterDataSpecUpdated() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	hooks := k.Hooks()
	// set query in collections, window at 100
	query := types.QueryMeta{
		RegistrySpecTimeframe: 100,
		QueryType:             "query",
		QueryId:               []byte("query"),
	}
	require.NoError(k.Query.Set(ctx, []byte("query"), query))

	// update spec to 50
	require.NoError(hooks.AfterDataSpecUpdated(ctx, "query", regtypes.DataSpec{
		ReportBufferWindow: 50,
	}))

	// check that spec is updated to 50
	meta, err := k.Query.Get(ctx, []byte("query"))
	require.NoError(err)
	require.EqualValues(meta.RegistrySpecTimeframe, time.Duration(50))
}
