package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (s *KeeperTestSuite) TestMsgProposeDispute() {
	require := s.Require()
	report := types.MicroReport{
		Reporter:  "trb1auznue6n56c0ptmmq7vydst8a0vyluje3q6dgn",
		Qid:       "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}

	fee := sdk.NewCoin("trb", sdk.NewInt(10000))

	msg := types.MsgProposeDispute{
		Creator:         Addr.String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             fee,
		PayFromBond:     false,
	}
	addy, _ := sdk.AccAddressFromBech32(Addr.String())
	s.bankKeeper.On("HasBalance", mock.Anything, mock.Anything, mock.Anything).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	val, _ := stakingtypes.NewValidator(sdk.ValAddress(addy), PubKey, stakingtypes.Description{Moniker: "test"})
	val.Jailed = false
	val.Status = stakingtypes.Bonded
	val.Tokens = sdk.NewInt(1000000000000000000)
	s.stakingKeeper.On("GetValidator", mock.Anything, mock.Anything).Return(val, true)
	msgRes, err := s.msgServer.ProposeDispute(s.goCtx, &msg)
	require.Nil(err)
	require.NotNil(msgRes)
	openDisputesRes := s.disputeKeeper.GetOpenDisputeIds(s.ctx)
	require.NotNil(openDisputesRes)
	require.Len(openDisputesRes.Ids, 1)
	require.Equal(openDisputesRes.Ids, []uint64{0})
	disputeRes := s.disputeKeeper.GetDisputeById(s.ctx, 0)
	require.NotNil(disputeRes)
	require.Equal(disputeRes.DisputeCategory, types.Warning)
	require.Equal(disputeRes.ReportEvidence.Reporter, "trb1auznue6n56c0ptmmq7vydst8a0vyluje3q6dgn")
}
