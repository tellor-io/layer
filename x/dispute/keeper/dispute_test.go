package keeper_test

import (
	"fmt"
	gomath "math"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func report() oracletypes.MicroReport {
	return oracletypes.MicroReport{
		Reporter:    sample.AccAddressBytes().String(),
		Power:       uint64(1),
		QueryId:     []byte{},
		Value:       "0x",
		Timestamp:   time.Unix(1696516597, 0),
		BlockNumber: 1,
	}
}

func (s *KeeperTestSuite) dispute() types.Dispute {
	report := report()
	hash := s.disputeKeeper.HashId(s.ctx, report, types.Warning)
	return types.Dispute{
		HashId:          hash[:],
		DisputeId:       1,
		DisputeCategory: types.Warning,
		DisputeFee:      math.ZeroInt(),
		InitialEvidence: report,
		Open:            true,
		SlashAmount:     math.NewInt(10000),
		BurnAmount:      math.NewInt(500),
	}
}

func (s *KeeperTestSuite) TestGetOpenDisputes() {
	res, err := s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.Nil(err)
	s.Equal([]uint64{}, res)

	dispute := s.dispute()
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	open, err := s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Len(open, 1)
	s.Equal([]uint64{dispute.DisputeId}, open)

	dispute.DisputeId = 2
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	open, err = s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Len(open, 2)
	s.Equal([]uint64{1, 2}, open)

	dispute.DisputeId = 3
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	open, err = s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Len(open, 3)
	s.Equal([]uint64{1, 2, 3}, open)

	s.NoError(s.disputeKeeper.CloseDispute(s.ctx, 2))
	open, err = s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Len(open, 2)
	s.Equal([]uint64{1, 3}, open)
}

func (s *KeeperTestSuite) TestGetDisputeByReporter() {
	dispute := s.dispute()
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))

	disputeByReporter, err := s.disputeKeeper.GetDisputeByReporter(s.ctx, dispute.InitialEvidence, types.Warning)
	s.NoError(err)
	s.Equal(dispute.DisputeId, disputeByReporter.DisputeId)

	dispute.DisputeId = 2
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))

	disputeByReporter, err = s.disputeKeeper.GetDisputeByReporter(s.ctx, dispute.InitialEvidence, types.Warning)
	s.NoError(err)
	s.Equal(uint64(2), disputeByReporter.DisputeId)
}

func (s *KeeperTestSuite) TestNextDisputeId() {
	dispute := s.dispute()
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	s.Equal(uint64(2), s.disputeKeeper.NextDisputeId(s.ctx))
	dispute.DisputeId = 2
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	s.Equal(uint64(3), s.disputeKeeper.NextDisputeId(s.ctx))
	dispute.DisputeId = 3
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	s.Equal(uint64(4), s.disputeKeeper.NextDisputeId(s.ctx))
	dispute.DisputeId = 4
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	s.Equal(uint64(5), s.disputeKeeper.NextDisputeId(s.ctx))
}

func (s *KeeperTestSuite) TestHashId() {
	hashId := s.disputeKeeper.HashId(s.ctx, report(), types.Warning)
	s.Len(hashId, 32)
}

func (s *KeeperTestSuite) TestReporterKey() {
	report := report()
	hash := s.disputeKeeper.HashId(s.ctx, report, types.Warning)
	expectedKey := fmt.Sprintf("%s:%x", report.Reporter, hash) // Replace with the expected key

	result := s.disputeKeeper.ReporterKey(s.ctx, report, types.Warning)

	s.Equal(expectedKey, result)
}

func (s *KeeperTestSuite) TestSetNewDispute() types.MsgProposeDispute {
	report := report()
	creator := sample.AccAddressBytes()
	disputeMsg := types.MsgProposeDispute{
		Creator:         creator.String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             sdk.NewCoin("loya", math.NewInt(1000)),
		PayFromBond:     false,
	}

	// mock dependency modules
	s.bankKeeper.On("HasBalance", s.ctx, sdk.MustAccAddressFromBech32(disputeMsg.Creator), disputeMsg.Fee).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", s.ctx, sdk.MustAccAddressFromBech32(disputeMsg.Creator), types.ModuleName, sdk.NewCoins(disputeMsg.Fee)).Return(nil)
	s.reporterKeeper.On("TotalReporterPower", s.ctx).Return(math.NewInt(1), nil)
	s.oracleKeeper.On("GetTotalTips", s.ctx).Return(math.NewInt(1), nil)

	s.NoError(s.disputeKeeper.SetNewDispute(s.ctx, creator, disputeMsg))

	return disputeMsg
}

func (s *KeeperTestSuite) TestSlashAndJailReporter() {
	s.ctx = s.ctx.WithBlockTime(time.Unix(1696516597, 0))
	report := report()
	dispute := s.dispute()
	reporterAcc := sdk.MustAccAddressFromBech32(report.Reporter)
	s.reporterKeeper.On("EscrowReporterStake", s.ctx, reporterAcc, report.Power, uint64(1), math.NewInt(10000), dispute.InitialEvidence.QueryId, dispute.HashId).Return(nil)
	s.reporterKeeper.On("JailReporter", s.ctx, reporterAcc, uint64(0)).Return(nil)
	s.oracleKeeper.On("FlagAggregateReport", s.ctx, report).Return(nil)
	s.NoError(s.disputeKeeper.SlashAndJailReporter(s.ctx, report, dispute.DisputeCategory, dispute.InitialEvidence.QueryId, dispute.HashId))
}

func (s *KeeperTestSuite) TestJailReporter() {
	reporterAcc := sample.AccAddressBytes()
	s.reporterKeeper.On("JailReporter", s.ctx, reporterAcc, uint64(0)).Return(nil)
	s.NoError(s.disputeKeeper.JailReporter(s.ctx, reporterAcc, uint64(0)))
}

func (s *KeeperTestSuite) TestGetSlashPercentageAndJailDuration() {
	testCases := []struct {
		name                    string
		cat                     types.DisputeCategory
		expectedSlashPercentage math.Int
		expectedJailTime        uint64
	}{
		{
			name:                    "Warning",
			cat:                     types.Warning,
			expectedSlashPercentage: math.NewInt(10000),
			expectedJailTime:        0,
		},
		{
			name:                    "Minor",
			cat:                     types.Minor,
			expectedSlashPercentage: math.NewInt(50000),
			expectedJailTime:        600,
		},
		{
			name:                    "Major",
			cat:                     types.Major,
			expectedSlashPercentage: math.NewInt(1000000),
			expectedJailTime:        gomath.MaxInt64,
		},
		{
			name: "Severe",
			cat:  4,
		},
	}
	s.oracleKeeper.On("FlagAggregateReport", s.ctx, report()).Return(nil)

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			slashAmount, jailTime, err := keeper.GetSlashPercentageAndJailDuration(tc.cat)
			if tc.name == "Severe" {
				s.Error(err, types.ErrInvalidDisputeCategory)
			} else {
				s.NoError(err)
				s.Equal(tc.expectedSlashPercentage, slashAmount)
				s.Equal(tc.expectedJailTime, jailTime)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetDisputeFee() {
	rep := report()
	rep.Power = 0
	disputeFee, err := s.disputeKeeper.GetDisputeFee(s.ctx, rep, types.Warning)
	s.NoError(err)
	s.Equal(math.ZeroInt(), disputeFee)

	rep.Power = 1
	disputeFee, err = s.disputeKeeper.GetDisputeFee(s.ctx, rep, types.Warning)
	s.NoError(err)
	s.Equal(math.NewInt(10000), disputeFee)

	rep.Power = 2
	disputeFee, err = s.disputeKeeper.GetDisputeFee(s.ctx, rep, types.Warning)
	s.NoError(err)
	s.Equal(math.NewInt(20000), disputeFee)

	rep.Power = 3
	disputeFee, err = s.disputeKeeper.GetDisputeFee(s.ctx, rep, types.Warning)
	s.NoError(err)
	s.Equal(math.NewInt(30000), disputeFee)

	rep.Power = 4
	disputeFee, err = s.disputeKeeper.GetDisputeFee(s.ctx, rep, types.Minor)
	s.NoError(err)
	s.Equal(math.NewInt(200000), disputeFee)

	// major
	disputeFee, err = s.disputeKeeper.GetDisputeFee(s.ctx, rep, types.Major)
	s.NoError(err)
	s.Equal(math.NewInt(4000000), disputeFee)
}

func (s *KeeperTestSuite) TestAddDisputeRound() {
	msg := s.TestSetNewDispute()
	sender := sdk.MustAccAddressFromBech32(msg.Creator)
	dispute, err := s.disputeKeeper.Disputes.Get(s.ctx, 1)
	s.NoError(err)

	dispute.DisputeStatus = types.Unresolved
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))

	fee := sdk.NewCoin("loya", math.NewInt(10))
	s.bankKeeper.On("HasBalance", s.ctx, sender, fee).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", s.ctx, sender, types.ModuleName, sdk.NewCoins(fee)).Return(nil)
	s.oracleKeeper.On("FlagAggregateReport", s.ctx, report()).Return(nil)
	s.NoError(s.disputeKeeper.AddDisputeRound(s.ctx, sender, dispute, msg))

	dispute1, err := s.disputeKeeper.Disputes.Get(s.ctx, 1)
	s.NoError(err)
	s.Equal(types.Unresolved, dispute1.DisputeStatus)
	s.True(!dispute1.Open)
	s.Equal(uint64(1), dispute1.DisputeRound)
	// attempt to start a new round for a closed dispute
	s.Error(s.disputeKeeper.AddDisputeRound(s.ctx, sender, dispute1, msg), "can't start a new round for this dispute 1; dispute closed")

	dispute2, err := s.disputeKeeper.Disputes.Get(s.ctx, 2)
	s.NoError(err)
	s.Equal(types.Voting, dispute2.DisputeStatus)
	s.Equal(dispute1.HashId, dispute2.HashId)
	s.True(dispute2.Open)
	s.Equal(uint64(2), dispute2.DisputeRound)
}

func (s *KeeperTestSuite) TestSetBlockInfo() {
	dispute := s.dispute()
	s.reporterKeeper.On("TotalReporterPower", s.ctx).Return(math.NewInt(1), nil)
	s.oracleKeeper.On("GetTotalTips", s.ctx).Return(math.NewInt(1), nil)
	s.NoError(s.disputeKeeper.SetBlockInfo(s.ctx, dispute.HashId))

	expectedBlockInfo := types.BlockInfo{
		TotalReporterPower: math.NewInt(1),
		TotalUserTips:      math.NewInt(1),
	}
	blockInfo, err := s.disputeKeeper.BlockInfo.Get(s.ctx, dispute.HashId)
	s.NoError(err)
	s.Equal(expectedBlockInfo, blockInfo)
}

func (s *KeeperTestSuite) TestCloseDispute() {
	dispute := s.dispute()
	s.NoError(s.disputeKeeper.Disputes.Set(s.ctx, dispute.DisputeId, dispute))
	open, err := s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Len(open, 1)

	s.NoError(s.disputeKeeper.CloseDispute(s.ctx, dispute.DisputeId))
	open, err = s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.Len(open, 0)

	dispute, err = s.disputeKeeper.Disputes.Get(s.ctx, dispute.DisputeId)
	s.NoError(err)
	s.False(dispute.Open)
}
