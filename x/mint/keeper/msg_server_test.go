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
