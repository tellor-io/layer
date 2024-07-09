package keeper_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) TestMsgUpdateTeam() {
	require := s.Require()
	k := s.disputeKeeper
	ctx := s.ctx
	msgServer := s.msgServer

	params, err := k.Params.Get(ctx)
	require.NoError(err)
	oldAddr := params.TeamAddress
	oldAccAddr := sdk.AccAddress(oldAddr)
	newAddr := sample.AccAddress()

	res, err := msgServer.UpdateTeam(ctx, &types.MsgUpdateTeam{
		CurrentTeamAddress: oldAccAddr.String(),
		NewTeamAddress:     newAddr,
	})
	require.NoError(err)
	require.NotNil(res)

	// try to call from invalid current address
	res, err = msgServer.UpdateTeam(ctx, &types.MsgUpdateTeam{
		CurrentTeamAddress: hex.EncodeToString(oldAddr),
		NewTeamAddress:     newAddr,
	})
	require.Error(err)
	require.Nil(res)

	// try to call from wrong current address
	wrongAddr := sample.AccAddressBytes()
	res, err = msgServer.UpdateTeam(ctx, &types.MsgUpdateTeam{
		CurrentTeamAddress: wrongAddr.String(),
		NewTeamAddress:     newAddr,
	})
	require.Error(err)
	require.Nil(res)

	// try to update to invalid Addr
	res, err = msgServer.UpdateTeam(ctx, &types.MsgUpdateTeam{
		CurrentTeamAddress: newAddr,
		NewTeamAddress:     hex.EncodeToString(oldAddr),
	})
	require.Error(err)
	require.Nil(res)
}
