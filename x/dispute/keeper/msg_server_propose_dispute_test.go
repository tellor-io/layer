package keeper_test

import (
	"encoding/hex"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// propose a single dispute on an array of reports
//
// extra field eveidence that is list of other reports
// iterate through those reports, see if it affects median, dispute those reports 
// geter for if bridge depots have been claimed, bridge deposit id as input
// getter for a bunch of dispute info 

func (s *KeeperTestSuite) TestMsgProposeDisputeFromAccount() sdk.AccAddress {
	addr := sample.AccAddressBytes()
	s.ctx = s.ctx.WithBlockTime(time.Now())
	qId, _ := hex.DecodeString("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992")
	report := oracletypes.MicroReport{
		Reporter:  addr.String(),
		QueryId:   qId,
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: time.Unix(1696516597, 0),
		Power:     1,
	}

	fee := sdk.NewCoin(layer.BondDenom, math.NewInt(10000))

	msg := types.MsgProposeDispute{
		Creator:         addr.String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             fee,
		PayFromBond:     false,
	}

	s.reporterKeeper.On("EscrowReporterStake", s.ctx, addr, uint64(1), uint64(0), math.NewInt(10_000), mock.Anything).Return(nil)
	s.reporterKeeper.On("TotalReporterPower", s.ctx).Return(math.NewInt(1), nil)
	s.oracleKeeper.On("GetTotalTips", s.ctx).Return(math.NewInt(1), nil)
	s.reporterKeeper.On("JailReporter", s.ctx, addr, uint64(0)).Return(nil)

	s.bankKeeper.On("HasBalance", s.ctx, addr, fee).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", s.ctx, addr, mock.Anything, sdk.NewCoins(fee)).Return(nil)
	s.oracleKeeper.On("FlagAggregateReport", s.ctx, report).Return(nil)

	msgRes, err := s.msgServer.ProposeDispute(s.ctx, &msg)
	s.NoError(err)
	s.NotNil(msgRes)
	openDisputesRes, err := s.disputeKeeper.GetOpenDisputes(s.ctx)
	s.NoError(err)
	s.NotNil(openDisputesRes)
	s.Len(openDisputesRes, 1)
	s.Equal(openDisputesRes, []uint64{1})
	disputeRes, err := s.disputeKeeper.Disputes.Get(s.ctx, 1)
	s.NoError(err)
	s.NotNil(disputeRes)
	s.Equal(disputeRes.DisputeCategory, types.Warning)
	s.Equal(disputeRes.ReportEvidence.Reporter, addr.String())
	s.Equal(disputeRes.DisputeStatus, types.Voting)
	return addr
}
