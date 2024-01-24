package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"

	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/tellor-io/layer/x/dispute/types"
)

func (s *KeeperTestSuite) TestMsgProposeDisputeFromAccount() {
	require := s.Require()
	report := types.MicroReport{
		Reporter:  "tellor1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34ds5rz",
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}

	fee := sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10000000000))

	msg := types.MsgProposeDispute{
		Creator:         Addr.String(),
		Report:          &report,
		DisputeCategory: types.Warning,
		Fee:             fee,
		PayFromBond:     false,
	}
	// addy, _ := sdk.AccAddressFromBech32(Addr.String())
	//sdk.ValAddress(addy)
	val, _ := stakingtypes.NewValidator(Addr.String(), PubKey, stakingtypes.Description{Moniker: "test"})
	val.Jailed = false
	val.Status = stakingtypes.Bonded
	val.Tokens = math.NewInt(1000000000)

	// mock dependency modules
	s.bankKeeper.On("HasBalance", mock.Anything, mock.Anything, mock.Anything).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.bankKeeper.On("SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.stakingKeeper.On("TokensFromConsensusPower", mock.Anything, mock.Anything).Return(math.NewInt(100))
	s.slashingKeeper.On("GetValidatorSigningInfo", mock.Anything, mock.Anything).Return(slashingtypes.ValidatorSigningInfo{}, nil)
	s.slashingKeeper.On("SetValidatorSigningInfo", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.stakingKeeper.On("Jail", mock.Anything, mock.Anything).Return(nil)
	s.stakingKeeper.On("RemoveValidatorTokens", mock.Anything, mock.Anything, mock.Anything).Return(val, nil)
	s.stakingKeeper.On("GetValidator", mock.Anything, mock.Anything).Return(val, nil)
	msgRes, err := s.msgServer.ProposeDispute(s.goCtx, &msg)
	require.NoError(err)
	require.NotNil(msgRes)
	openDisputesRes := s.disputeKeeper.GetOpenDisputeIds(s.ctx)
	require.NotNil(openDisputesRes)
	require.Len(openDisputesRes.Ids, 1)
	require.Equal(openDisputesRes.Ids, []uint64{1})
	disputeRes := s.disputeKeeper.GetDisputeById(s.ctx, 1)
	require.NotNil(disputeRes)
	require.Equal(disputeRes.DisputeCategory, types.Warning)
	require.Equal(disputeRes.ReportEvidence.Reporter, "tellor1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34ds5rz")
	require.Equal(disputeRes.DisputeStatus, types.Voting)
}
