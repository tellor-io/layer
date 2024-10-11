package keeper_test

import "github.com/tellor-io/layer/x/dispute/types"

func (s *KeeperTestSuite) TestAddEvidence() {
	require := s.Require()
	require.NotNil(s.msgServer)

	testCases := []struct {
		name  string
		msg   *types.MsgAddEvidence
		err   bool
		setup func()
	}{
		{
			name: "empty message",
			msg:  &types.MsgAddEvidence{},
			err:  true,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.setup != nil {
				tc.setup()
			}
			_, err := s.msgServer.AddEvidence(s.ctx, tc.msg)
			if tc.err {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
