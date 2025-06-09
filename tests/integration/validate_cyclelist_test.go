package integration_test

import (
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
)

func (s *IntegrationTestSuite) TestValidateCycleList() {
	require := s.Require()
	s.Setup.Ctx = s.Setup.Ctx.WithConsensusParams(cmtproto.ConsensusParams{
		Block: &cmtproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   100_000_000,
		},
		Evidence: &cmtproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
			MaxBytes:        10000,
		},
		Validator: &cmtproto.ValidatorParams{
			PubKeyTypes: []string{
				cmttypes.ABCIPubKeyTypeEd25519,
			},
		},
		Abci: &cmtproto.ABCIParams{
			VoteExtensionsEnableHeight: 1,
		},
	})
	_, vals, _ := s.Setup.CreateValidators(1)
	for _, val := range vals {
		err := s.Setup.Bridgekeeper.SetEVMAddressByOperator(s.Setup.Ctx, val.String(), []byte("not real"))
		s.NoError(err)
	}
	//---------------------------------------------------------------------------
	// Height 1 - eth gets 3 blocks to start
	//---------------------------------------------------------------------------\
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, err := s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	cycle1, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes := ethQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle1)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(1))
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 2 - eth
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(2))
	cycle1, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = ethQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle1)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 3 - eth final block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(3))
	cycle1, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = ethQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle1)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 4 - btc first block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(4))
	cycle2, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = btcQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle2)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 5 - btc final block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(5))
	cycle2, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = btcQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle2)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 6 - trb first block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(6))
	cycle2, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = trbQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle2)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 7 - trb final block
	//---------------------------------------------------------------------------

	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(7))
	cycle3, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = trbQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle3)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 8 - eth first block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(8))
	cycle3, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = ethQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle3)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	//---------------------------------------------------------------------------
	// Height 9 - eth final block
	//---------------------------------------------------------------------------
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
	_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
	require.NoError(err)
	require.Equal(s.Setup.Ctx.BlockHeight(), int64(9))
	cycle2, err = s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
	require.NoError(err)
	queryDataBytes = ethQueryData
	require.NoError(err)
	require.Equal(queryDataBytes, cycle2)
	_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
	require.NoError(err)

	// loop through 20 more blocks
	list, err := s.Setup.Oraclekeeper.GetCyclelist(s.Setup.Ctx)
	require.NoError(err)
	for i := 0; i < 20; i++ {
		s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(s.Setup.Ctx.BlockHeight() + 1)
		_, err = s.Setup.App.BeginBlocker(s.Setup.Ctx)
		require.NoError(err)

		query, err := s.Setup.Oraclekeeper.GetCurrentQueryInCycleList(s.Setup.Ctx)
		require.NoError(err)
		require.Contains(list, query)

		_, err = s.Setup.App.EndBlocker(s.Setup.Ctx)
		require.NoError(err)
	}
}
