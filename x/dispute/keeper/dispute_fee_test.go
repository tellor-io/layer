package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	layer "github.com/tellor-io/layer/types"
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

	s.reporterKeeper.On("FeefromReporterStake", s.ctx, addr, math.OneInt()).Return(nil)
	err := s.disputeKeeper.PayFromBond(s.ctx, addr, sdk.NewCoin(layer.BondDenom, math.NewInt(1)))
	s.Nil(err)
}
