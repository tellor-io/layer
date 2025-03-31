package dispute_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/mocks"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

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
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(24 * time.Hour))

	// check with no open disputes
	err := dispute.CheckOpenDisputesForExpiration(ctx, k)
	require.NoError(err)

	// check with open dispute
	require.NoError(k.Disputes.Set(ctx, 1, types.Dispute{
		DisputeId:        1,
		DisputeStatus:    types.Prevote,
		DisputeStartTime: ctx.BlockTime().Add(-time.Hour),
		DisputeEndTime:   ctx.BlockTime().Add(time.Hour),
		Open:             true,
	}))

	err = dispute.CheckOpenDisputesForExpiration(ctx, k)
	require.NoError(err)

	// check again after endtime passes
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(2 * time.Hour))
	err = dispute.CheckOpenDisputesForExpiration(ctx, k)
	require.NoError(err)
	dispute, err := k.Disputes.Get(ctx, 1)
	require.NoError(err)
	require.Equal(dispute.DisputeStatus, types.Failed)
	require.Equal(dispute.Open, false)
}

func (s *TestSuite) TestEndBlocker() {
	require := require.New(s.T())
	k := s.disputeKeeper
	ctx := s.ctx

	ctx = ctx.WithBlockHeight(10)
	ctx = ctx.WithBlockTime(time.Now())

	err := dispute.EndBlocker(ctx, k)
	require.NoError(err)

}

func BenchmarkDisputeEndBlocker(b *testing.B) {
	disputeKeeper, ok, rk, _, _, ctx := keepertest.DisputeKeeper(b)
	ctx = ctx.WithBlockHeight(10)
	ctx = ctx.WithBlockTime(time.Now())

	testCases := []struct {
		name          string
		disputeCount  int
		matchingBlock bool
	}{
		{"No Disputes", 0, true},
		{"1 Dispute At Current Block", 1, true},
		{"1 Dispute At Different Block", 1, false},
		{"10 Disputes At Current Block", 10, true},
		{"10 Disputes At Different Block", 10, false},
		{"100 Disputes At Current Block", 100, true},
	}

	for _, tc := range testCases {
		fmt.Println("running: ", tc.name)
		b.Run(tc.name, func(b *testing.B) {
			// use variable block heights
			ctx := ctx.WithBlockHeight(int64(b.N))
			blockHeight := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())

			// mocks
			rk.On("TotalReporterPower", ctx).Return(math.NewInt(1000000000), nil)
			ok.On("GetTotalTips", ctx).Return(math.NewInt(1000000000), nil)

			// set disputes at current block or block before
			// EndBlock looks for disputes opened this block
			for i := range make([]int, tc.disputeCount) {
				disputeHeight := blockHeight
				if !tc.matchingBlock {
					disputeHeight = blockHeight - 1
				}

				testDispute := types.Dispute{
					HashId:            fmt.Appendf(nil, "testHash%d", i),
					BlockNumber:       disputeHeight,
					Open:              true,
					DisputeId:         uint64(i),
					DisputeCategory:   types.Warning,
					DisputeStatus:     types.Prevote,
					DisputeStartTime:  ctx.BlockTime().Add(-time.Hour),
					DisputeEndTime:    ctx.BlockTime().Add(time.Hour),
					DisputeStartBlock: disputeHeight,
					SlashAmount:       math.NewInt(1000),
					BurnAmount:        math.NewInt(1000),
					InitialEvidence: oracletypes.MicroReport{
						Reporter:        fmt.Sprintf("testReport%d", i),
						Power:           uint64(1000),
						QueryId:         fmt.Appendf(nil, "testQuery%d", i),
						QueryType:       "SpotPrice",
						AggregateMethod: "weighted-median",
						Timestamp:       ctx.BlockTime().Add(-time.Hour),
						Value:           math.NewInt(1000).String(),
						Cyclelist:       false,
						BlockNumber:     disputeHeight,
						MetaId:          uint64(i),
					},
				}

				if err := disputeKeeper.Disputes.Set(ctx, testDispute.DisputeId, testDispute); err != nil {
					fmt.Println("error setting dispute", err)
					b.Fatal(err)
				}
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := dispute.EndBlocker(ctx, disputeKeeper); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDisputeBeginBlocker(b *testing.B) {
	disputeKeeper, _, rk, _, bk, ctx := keepertest.DisputeKeeper(b)

	testCases := []struct {
		name                 string
		openDisputeCount     int
		expiringDisputeCount int
	}{
		{"No Disputes", 0, 0},
		{"1 Open Dispute", 1, 0},
		{"1 Expiring Dispute", 0, 1},
		{"1 Open and 1 Expiring Dispute", 1, 1},
		{"10 Open Disputes", 10, 0},
		{"10 Expiring Disputes", 0, 10},
		{"10 Open and 10 Expiring Disputes", 10, 10},
	}

	// pre-populate all test cases
	testData := make(map[string]struct {
		ctx          sdk.Context
		keeper       keeper.Keeper
		openDisputes int
		execDisputes int
	})

	for _, tc := range testCases {
		// Create a fresh context for each test case
		testCtx := ctx.WithBlockHeight(10).WithBlockTime(time.Now())

		// Set up disputes for this test case
		for i := 0; i < tc.openDisputeCount; i++ {
			fmt.Println("creating open dispute", i)
			createOpenExpiringDispute(testCtx, disputeKeeper, i)
		}
		for i := 0; i < tc.expiringDisputeCount; i++ {
			fmt.Println("creating closed dispute", i)
			createClosedDisputeForExecution(testCtx, disputeKeeper, i)
		}

		// store the prepared test case
		testData[tc.name] = struct {
			ctx          sdk.Context
			keeper       keeper.Keeper
			openDisputes int
			execDisputes int
		}{
			ctx:          testCtx,
			keeper:       disputeKeeper,
			openDisputes: tc.openDisputeCount,
			execDisputes: tc.expiringDisputeCount,
		}
	}

	// mocks
	bk.On("BurnCoins", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rk.On("ReturnSlashedTokens", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)
	bk.On("SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	rk.On("UpdateJailedUntilOnFailedDispute", mock.Anything, mock.Anything).Return(nil)

	// run benchmarks
	for _, tc := range testCases {
		data := testData[tc.name]

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := dispute.BeginBlocker(data.ctx, data.keeper); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func createOpenExpiringDispute(ctx sdk.Context, disputeKeeper keeper.Keeper, i int) {
	currentTime := ctx.BlockTime()
	currentBlock := ctx.BlockHeight()
	reporter := sample.AccAddressBytes()

	// set vote end time to before block time and vote result to VoteResult_NO_TALLY
	disputeKeeper.Votes.Set(ctx, uint64(i), types.Vote{
		Id:         uint64(i),
		VoteStart:  currentTime.Add(-25 * time.Hour),
		VoteEnd:    currentTime.Add(-time.Hour),
		VoteResult: types.VoteResult_INVALID,
	})

	disputeKeeper.Disputes.Set(ctx, uint64(i), types.Dispute{
		HashId:            fmt.Appendf(nil, "testHash%d", i),
		BlockNumber:       uint64(currentBlock - 1),
		Open:              true,
		DisputeId:         uint64(i),
		DisputeCategory:   types.Warning,
		DisputeStatus:     types.Resolved,
		DisputeStartTime:  currentTime.Add(-time.Hour),
		DisputeEndTime:    currentTime.Add(time.Hour),
		DisputeStartBlock: uint64(currentBlock - 1),
		SlashAmount:       math.NewInt(1000),
		BurnAmount:        math.NewInt(1000),
		InitialEvidence: oracletypes.MicroReport{
			Reporter:        reporter.String(),
			Power:           uint64(1000),
			QueryId:         fmt.Appendf(nil, "testQuery%d", i),
			QueryType:       "SpotPrice",
			AggregateMethod: "weighted-median",
			Timestamp:       currentTime.Add(-time.Hour),
			Value:           math.NewInt(1000).String(),
			Cyclelist:       false,
			BlockNumber:     uint64(currentBlock - 2),
			MetaId:          uint64(i),
		},
	})
}

func createClosedDisputeForExecution(ctx sdk.Context, disputeKeeper keeper.Keeper, i int) {
	currentTime := ctx.BlockTime()
	currentBlock := ctx.BlockHeight()

	reporter := sample.AccAddressBytes()

	disputeKeeper.Disputes.Set(ctx, uint64(i), types.Dispute{
		DisputeId:         uint64(i),
		DisputeStatus:     types.Resolved,
		DisputeStartTime:  currentTime.Add(-25 * time.Hour),
		DisputeEndTime:    currentTime.Add(-time.Hour),
		Open:              false,
		DisputeStartBlock: 1,
		SlashAmount:       math.NewInt(1000),
		BurnAmount:        math.NewInt(1000),
		PendingExecution:  true,
		InitialEvidence: oracletypes.MicroReport{
			Reporter:        reporter.String(),
			Power:           uint64(1000),
			QueryId:         fmt.Appendf(nil, "testQuery%d", i),
			QueryType:       "SpotPrice",
			AggregateMethod: "weighted-median",
			Timestamp:       currentTime.Add(-time.Hour),
			Value:           math.NewInt(1000).String(),
			Cyclelist:       false,
			BlockNumber:     uint64(currentBlock - 1),
			MetaId:          uint64(i),
		},
	})

	// Add missing vote creation
	disputeKeeper.Votes.Set(ctx, uint64(i), types.Vote{
		Id:         uint64(i),
		VoteStart:  currentTime.Add(-25 * time.Hour),
		VoteEnd:    currentTime.Add(-time.Hour),
		VoteResult: types.VoteResult_INVALID,
	})
}
