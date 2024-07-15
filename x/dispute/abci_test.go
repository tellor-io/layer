package dispute_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	"github.com/tellor-io/layer/testutil"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/mocks"
	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestSuite struct {
	suite.Suite

	ctx           sdk.Context
	disputeKeeper keeper.Keeper

	accountKeeper  *mocks.AccountKeeper
	bankKeeper     *mocks.BankKeeper
	oracleKeeper   *mocks.OracleKeeper
	reporterKeeper *mocks.ReporterKeeper
}

func (s *TestSuite) SetupTest() {
	config.SetupConfig()

	s.disputeKeeper,
		s.oracleKeeper,
		s.reporterKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.ctx = keepertest.DisputeKeeper(s.T())
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestBeginBlocker() {
	require := require.New(s.T())
	k := s.disputeKeeper
	ctx := s.ctx

	err := dispute.BeginBlocker(ctx, k)
	require.NoError(err)
}

func (s *TestSuite) TestCheckPrevoteDisputesForExpiration() {
	require := require.New(s.T())
	k := s.disputeKeeper
	ctx := s.ctx
	ctx = testutil.WithBlockTime(s.ctx, ctx.HeaderInfo().Time.Add(24*time.Hour))

	// check with no open disputes
	err := dispute.CheckPrevoteDisputesForExpiration(ctx, k)
	require.NoError(err)

	// check with open dispute
	require.NoError(k.Disputes.Set(ctx, 1, types.Dispute{
		DisputeId:        1,
		DisputeStatus:    types.Prevote,
		DisputeStartTime: ctx.HeaderInfo().Time.Add(-time.Hour),
		DisputeEndTime:   ctx.HeaderInfo().Time.Add(time.Hour),
		Open:             true,
	}))

	err = dispute.CheckPrevoteDisputesForExpiration(ctx, k)
	require.NoError(err)

	// check again after endtime passes
	ctx = testutil.WithBlockTime(s.ctx, ctx.HeaderInfo().Time.Add(2*time.Hour))
	err = dispute.CheckPrevoteDisputesForExpiration(ctx, k)
	require.NoError(err)
	dispute, err := k.Disputes.Get(ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeStatus, types.Failed)
	require.Equal(dispute.Open, false)
}
