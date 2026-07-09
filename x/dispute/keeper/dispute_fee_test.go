package keeper_test

import (
	"time"

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

	s.reporterKeeper.On("FeefromReporterStake", s.ctx, addr, math.OneInt(), []byte("hash"), true).Return(nil)
	err := s.disputeKeeper.PayFromBond(s.ctx, addr, sdk.NewCoin(layer.BondDenom, math.NewInt(1)), []byte("hash"), true)
	s.Nil(err)
}

func (s *KeeperTestSuite) TestPayDisputeFee() {
	acct := sample.AccAddressBytes()
	fee := sdk.NewCoin(layer.BondDenom, math.OneInt())
	s.bankKeeper.On("HasBalance", s.ctx, acct, fee).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", s.ctx, acct, types.ModuleName, sdk.NewCoins(fee)).Return(nil)
	// from account
	s.NoError(s.disputeKeeper.PayDisputeFee(s.ctx, acct, fee, false, []byte("hash"), true))
	// from bond
	s.reporterKeeper.On("FeefromReporterStake", s.ctx, acct, math.OneInt(), []byte("hash"), true).Return(nil)
	s.NoError(s.disputeKeeper.PayDisputeFee(s.ctx, acct, fee, true, []byte("hash"), true))
}

func (k *KeeperTestSuite) TestReturnSlashedTokens() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)
	// reporter keeper returns per-pool amounts: full slash amount is bonded here.
	k.reporterKeeper.On("ReturnSlashedTokens", k.ctx, dispute.HashId, math.ZeroInt()).Return(dispute.SlashAmount, math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, dispute.SlashAmount))).Return(nil)
	k.NoError(k.disputeKeeper.ReturnSlashedTokens(k.ctx, dispute, math.ZeroInt()))
}

func (k *KeeperTestSuite) TestReturnFeetoStake() {
	feePayer := sample.AccAddressBytes()
	k.reporterKeeper.On("FeeRefund", k.ctx, []byte("hash"), feePayer, math.OneInt()).Return(math.OneInt(), math.ZeroInt(), nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, math.OneInt()))).Return(nil)
	k.NoError(k.disputeKeeper.ReturnFeetoStake(k.ctx, []byte("hash"), feePayer, math.OneInt()))
}

// TestReturnSlashedTokensMixedPools guards mixed-pool routing: when the
// reporter keeper returns both bonded and unbonded refund amounts, the dispute
// caller must route each to its own pool instead of sending the full slash
// amount to a single pool.
func (k *KeeperTestSuite) TestReturnSlashedTokensMixedPools() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)
	bondedPortion := math.NewInt(6000)
	unbondedPortion := dispute.SlashAmount.Sub(bondedPortion) // 4000
	k.reporterKeeper.On("ReturnSlashedTokens", k.ctx, dispute.HashId, math.ZeroInt()).Return(bondedPortion, unbondedPortion, nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, bondedPortion))).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, unbondedPortion))).Return(nil)
	k.NoError(k.disputeKeeper.ReturnSlashedTokens(k.ctx, dispute, math.ZeroInt()))
}

// TestReturnSlashedTokensRoutesReporterPoolAmountsExactly guards the caller
// boundary: the dispute keeper only moves the exact per-pool amounts returned
// by the reporter keeper. Proportional dust assignment belongs inside reporter
// keeper before these pool amounts are reported.
func (k *KeeperTestSuite) TestReturnSlashedTokensRoutesReporterPoolAmountsExactly() {
	k.ctx = k.ctx.WithBlockTime(time.Now())
	dispute := k.dispute(k.ctx)
	bondedPortion := math.NewInt(5999)
	unbondedPortion := math.NewInt(3999)
	k.reporterKeeper.On("ReturnSlashedTokens", k.ctx, dispute.HashId, math.ZeroInt()).Return(bondedPortion, unbondedPortion, nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, bondedPortion))).Return(nil)
	k.bankKeeper.On("SendCoinsFromModuleToModule", k.ctx, types.ModuleName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin(layer.BondDenom, unbondedPortion))).Return(nil)
	k.NoError(k.disputeKeeper.ReturnSlashedTokens(k.ctx, dispute, math.ZeroInt()))
	k.reporterKeeper.AssertExpectations(k.T())
	k.bankKeeper.AssertExpectations(k.T())
}
