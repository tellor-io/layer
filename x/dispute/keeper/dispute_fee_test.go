package keeper_test

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/mock"
)

func (s *KeeperTestSuite) TestPayFromAccount() {
	require := s.Require()
	s.bankKeeper.On("HasBalance", mock.Anything, mock.Anything, mock.Anything).Return(true)
	s.bankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err := s.disputeKeeper.PayFromAccount(s.ctx, Addr, sdk.NewCoin("trb", math.NewInt(1)))
	require.Nil(err)
}

func (s *KeeperTestSuite) TestPayFromBond() {
	require := s.Require()

	val, _ := stakingtypes.NewValidator(Addr.String(), PubKey, stakingtypes.Description{Moniker: "test"})
	val.Jailed = false
	val.Status = stakingtypes.Bonded
	val.Tokens = math.NewInt(1)
	s.stakingKeeper.On("GetValidator", mock.Anything, mock.Anything).Return(val, nil)
	s.stakingKeeper.On("RemoveValidatorTokens", mock.Anything, mock.Anything, mock.Anything).Return(val, nil)
	s.bankKeeper.On("SendCoinsFromModuleToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	err := s.disputeKeeper.PayFromBond(s.ctx, Addr, sdk.NewCoin("trb", math.NewInt(1)))
	require.Nil(err)
	// Should error since fee is more than bond
	err = s.disputeKeeper.PayFromBond(s.ctx, Addr, sdk.NewCoin("trb", math.NewInt(10)))
	require.NotNil(err)
}
