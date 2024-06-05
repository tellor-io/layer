package keeper_test

import (
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
	"github.com/tellor-io/layer/x/dispute/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *KeeperTestSuite) TestPayFromAccount() {
	addr := sample.AccAddressBytes()

	s.bankKeeper.On("HasBalance", s.ctx, addr, sdk.NewCoin(layer.BondDenom, math.NewInt(1))).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := s.disputeKeeper.PayFromAccount(s.ctx, addr, sdk.NewCoin(layer.BondDenom, math.NewInt(1)))
	s.Nil(err)
}

func (s *KeeperTestSuite) TestPayFromBond() {
	addr := sample.AccAddressBytes()

	s.reporterKeeper.On("FeefromReporterStake", s.ctx, addr, math.OneInt(), []byte("hash")).Return(nil)
	err := s.disputeKeeper.PayFromBond(s.ctx, addr, sdk.NewCoin(layer.BondDenom, math.NewInt(1)), []byte("hash"))
	s.Nil(err)
}

func (s *KeeperTestSuite) TestPayDisputeFee() {
	acct := sample.AccAddressBytes()
	fee := sdk.NewCoin(layer.BondDenom, math.OneInt())
	s.bankKeeper.On("HasBalance", s.ctx, acct, fee).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", s.ctx, acct, types.ModuleName, sdk.NewCoins(fee)).Return(nil)
	// from account
	s.NoError(s.disputeKeeper.PayDisputeFee(s.ctx, acct, fee, false, []byte("hash")))
	// from bond
	s.reporterKeeper.On("FeefromReporterStake", s.ctx, acct, math.OneInt(), []byte("hash")).Return(nil)
	s.NoError(s.disputeKeeper.PayDisputeFee(s.ctx, acct, fee, true, []byte("hash")))
}

func (k *KeeperTestSuite) TestReturnSlashedTokens() {
	dispute := k.dispute()
	k.reporterKeeper.On("ReturnSlashedTokens", k.ctx, dispute.ReportEvidence.Reporter, dispute.SlashAmount, dispute.HashId).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, dispute.SlashAmount))).Return(nil)
	k.NoError(k.disputeKeeper.ReturnSlashedTokens(k.ctx, dispute))
}

func (k *KeeperTestSuite) TestReturnFeetoStake() {
	repAcc := sample.AccAddressBytes()
	k.reporterKeeper.On("FeeRefund", k.ctx, repAcc, []byte("hash"), math.OneInt()).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, math.OneInt()))).Return(nil)
	k.NoError(k.disputeKeeper.ReturnFeetoStake(k.ctx, repAcc, []byte("hash"), math.OneInt()))
}
