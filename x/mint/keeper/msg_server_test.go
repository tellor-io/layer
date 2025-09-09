package keeper_test

import (
	"github.com/tellor-io/layer/x/mint/keeper"
	"github.com/tellor-io/layer/x/mint/types"
)

func (s *KeeperTestSuite) TestNewMsgServerImpl() {
	require := s.Require()
	k := s.mintKeeper

	msgServer := keeper.NewMsgServerImpl(k)
	require.NotNil(msgServer)
}

func (s *KeeperTestSuite) TestInit() {
	require := s.Require()
	k := s.mintKeeper

	msgServer := keeper.NewMsgServerImpl(k)
	require.NotNil(msgServer)

	// call with empty authority
	res, err := msgServer.Init(s.ctx, &types.MsgInit{})
	require.Error(err)
	require.Nil(res)

	// call with good authority
	require.NoError(k.Minter.Set(s.ctx, types.DefaultMinter()))
	res, err = msgServer.Init(s.ctx, &types.MsgInit{
		Authority: k.GetAuthority(),
	})
	require.NoError(err)
	require.NotNil(res)
}

func (s *KeeperTestSuite) TestUpdateExtraRewardRate() {
	require := s.Require()
	k := s.mintKeeper
	msgServer := keeper.NewMsgServerImpl(k)

	initialParams := types.ExtraRewardParams{
		DailyExtraRewards: 1000000,
		PreviousBlockTime: nil,
		BondDenom:         types.DefaultBondDenom,
	}
	err := k.ExtraRewardParams.Set(s.ctx, initialParams)
	require.NoError(err)

	// Invalid authority should fail
	invalidMsg := &types.MsgUpdateExtraRewardRate{
		Authority:         "invalid_authority",
		DailyExtraRewards: 2000000,
		BondDenom:         types.DefaultBondDenom,
	}
	res, err := msgServer.UpdateExtraRewardRate(s.ctx, invalidMsg)
	require.Error(err)
	require.Contains(err.Error(), "invalid authority")
	require.Nil(res)

	// Valid authority with positive reward rate should succeed
	validMsg := &types.MsgUpdateExtraRewardRate{
		Authority:         k.GetAuthority(),
		DailyExtraRewards: 5000000,
		BondDenom:         types.DefaultBondDenom,
	}
	res, err = msgServer.UpdateExtraRewardRate(s.ctx, validMsg)
	require.NoError(err)
	require.NotNil(res)

	// Verify params were updated
	updatedParams, err := k.ExtraRewardParams.Get(s.ctx)
	require.NoError(err)
	require.Equal(int64(5000000), updatedParams.DailyExtraRewards)

	// Zero reward rate should fail
	zeroMsg := &types.MsgUpdateExtraRewardRate{
		Authority:         k.GetAuthority(),
		DailyExtraRewards: 0,
		BondDenom:         types.DefaultBondDenom,
	}
	res, err = msgServer.UpdateExtraRewardRate(s.ctx, zeroMsg)
	require.Error(err)
	require.Contains(err.Error(), "daily extra rewards must be positive")
	require.Nil(res)

	// Negative reward rate should fail
	negativeMsg := &types.MsgUpdateExtraRewardRate{
		Authority:         k.GetAuthority(),
		DailyExtraRewards: -1000,
		BondDenom:         types.DefaultBondDenom,
	}
	res, err = msgServer.UpdateExtraRewardRate(s.ctx, negativeMsg)
	require.Error(err)
	require.Contains(err.Error(), "daily extra rewards must be positive")
	require.Nil(res)

	// Verify that updating non-existent params works (with default initialization)
	// The implementation will use GetExtraRewardRateParams which returns defaults when not found
	newMsg := &types.MsgUpdateExtraRewardRate{
		Authority:         k.GetAuthority(),
		DailyExtraRewards: 3000000,
		BondDenom:         "newdenom",
	}
	res, err = msgServer.UpdateExtraRewardRate(s.ctx, newMsg)
	require.NoError(err)
	require.NotNil(res)

	// Verify params were updated
	newParams, err := k.ExtraRewardParams.Get(s.ctx)
	require.NoError(err)
	require.Equal(int64(3000000), newParams.DailyExtraRewards)

	// Verify previous block time is preserved during update
	currentTime := s.ctx.BlockTime()
	paramsWithTime := types.ExtraRewardParams{
		DailyExtraRewards: 1000000,
		PreviousBlockTime: &currentTime,
		BondDenom:         types.DefaultBondDenom,
	}
	err = k.ExtraRewardParams.Set(s.ctx, paramsWithTime)
	require.NoError(err)

	updateMsg := &types.MsgUpdateExtraRewardRate{
		Authority:         k.GetAuthority(),
		DailyExtraRewards: 7000000,
		BondDenom:         types.DefaultBondDenom,
	}
	res, err = msgServer.UpdateExtraRewardRate(s.ctx, updateMsg)
	require.NoError(err)
	require.NotNil(res)

	// Verify previous block time was preserved
	finalParams, err := k.ExtraRewardParams.Get(s.ctx)
	require.NoError(err)
	require.Equal(int64(7000000), finalParams.DailyExtraRewards)
	require.NotNil(finalParams.PreviousBlockTime)
	require.Equal(currentTime, *finalParams.PreviousBlockTime)
}
