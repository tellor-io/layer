package keeper_test

import (
	"encoding/hex"

	"github.com/tellor-io/layer/x/oracle/types"
	regtypes "github.com/tellor-io/layer/x/registry/types"
)

func (s *KeeperTestSuite) TestMsgUpdateCycleList() {
	require := s.Require()
	ctx := s.ctx
	k := s.oracleKeeper
	regK := s.registryKeeper

	// bad authority
	req := types.MsgUpdateCyclelist{
		Authority: "bad",
	}
	_, err := s.msgServer.UpdateCyclelist(ctx, &req)
	require.ErrorContains(err, "invalid authority")

	// good authority
	matic, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000056D6174696300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	req = types.MsgUpdateCyclelist{
		Authority: k.GetAuthority(),
		Cyclelist: [][]byte{matic},
	}
	regK.On("GetSpec", ctx, "SpotPrice").Return(regtypes.DataSpec{}, nil)
	_, err = s.msgServer.UpdateCyclelist(ctx, &req)
	require.NoError(err)

	cyclelist, err := k.GetCyclelist(ctx)
	require.NoError(err)
	require.Equal([][]byte{matic}, cyclelist)

	req = types.MsgUpdateCyclelist{
		Authority: k.GetAuthority(),
		Cyclelist: make([][]byte, 0),
	}

	_, err = s.msgServer.UpdateCyclelist(ctx, &req)
	require.ErrorContains(err, "cyclelist is empty")
}
