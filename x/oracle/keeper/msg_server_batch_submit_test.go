package keeper_test

import (
	"encoding/hex"
	"errors"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
)

func (s *KeeperTestSuite) TestBatchSubmitValue() {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.reporterKeeper

	// Setup test data
	addr := sample.AccAddressBytes()
	addr2 := sample.AccAddressBytes()
	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)

	// Create another query data for testing multiple submissions
	qData2 := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	qDataBz2, err := utils.QueryBytesFromString(qData2)
	require.NoError(err)
	queryId2 := utils.QueryIDFromData(qDataBz2)

	// Setup queries in storage
	query1 := types.QueryMeta{
		Id:                      1,
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               qDataBz,
		QueryType:               "SpotPrice",
		CycleList:               true,
	}

	query2 := types.QueryMeta{
		Id:                      2,
		Amount:                  math.NewInt(200_000),
		Expiration:              25,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               qDataBz2,
		QueryType:               "SpotPrice",
		CycleList:               true,
	}

	err = k.Query.Set(s.ctx, collections.Join(queryId, query1.Id), query1)
	require.NoError(err)
	err = k.Query.Set(s.ctx, collections.Join(queryId2, query2.Id), query2)
	require.NoError(err)
	err = k.QueryDataLimit.Set(s.ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)

	params, err := k.Params.Get(s.ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount

	s.ctx = s.ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter()).WithBlockHeight(18)

	// Test 1: Successful batch submission
	s.Run("Successful batch submission", func() {
		// Mock registry keeper for spec lookups
		s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(spotSpec, nil).Times(2)

		rk.On("ReporterStake", s.ctx, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

		msg := &types.MsgBatchSubmitValue{
			Creator: addr.String(),
			Values: []*types.SubmitValueItem{
				{
					QueryData: qDataBz,
					Value:     "000000000000000000000000000000000000000000000000000000000000001e",
				},
				{
					QueryData: qDataBz2,
					Value:     "000000000000000000000000000000000000000000000000000000000000002a",
				},
			},
		}

		res, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.NoError(err)
		require.NotNil(res)
		require.Empty(res.FailedIndices)

		// Verify reports were stored
		iter1, err := k.Reports.Indexes.IdQueryId.MatchExact(s.ctx, collections.Join(query1.Id, queryId))
		require.NoError(err)
		reports1, err := iter1.FullKeys()
		require.NoError(err)
		require.Len(reports1, 1)

		iter2, err := k.Reports.Indexes.IdQueryId.MatchExact(s.ctx, collections.Join(query2.Id, queryId2))
		require.NoError(err)
		reports2, err := iter2.FullKeys()
		require.NoError(err)
		require.Len(reports2, 1)
	})

	// Test 2: Invalid creator address
	s.Run("Invalid creator address", func() {
		msg := &types.MsgBatchSubmitValue{
			Creator: "invalid_address",
			Values: []*types.SubmitValueItem{
				{
					QueryData: qDataBz,
					Value:     "000000000000000000000000000000000000000000000000000000000000001e",
				},
			},
		}

		_, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.Error(err)
		require.Contains(err.Error(), "invalid creator address")
	})

	// Test 3: Batch exceeds max size
	s.Run("Batch exceeds max size", func() {
		// Create 21 items (max is hardcoded to 20)
		values := make([]*types.SubmitValueItem, 21)
		for i := range values {
			values[i] = &types.SubmitValueItem{
				QueryData: qDataBz,
				Value:     "000000000000000000000000000000000000000000000000000000000000001e",
			}
		}

		msg := &types.MsgBatchSubmitValue{
			Creator: addr.String(),
			Values:  values,
		}

		_, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.Error(err)
		require.Contains(err.Error(), "too many reports in batch")
	})

	// Test 4: Insufficient reporter stake
	s.Run("Insufficient reporter stake", func() {
		rk.On("ReporterStake", s.ctx, addr, queryId).Return(minStakeAmt.Sub(math.NewInt(100)), nil).Once()

		msg := &types.MsgBatchSubmitValue{
			Creator: addr.String(),
			Values: []*types.SubmitValueItem{
				{
					QueryData: qDataBz,
					Value:     "000000000000000000000000000000000000000000000000000000000000001e",
				},
			},
		}

		_, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.Error(err)
		require.Contains(err.Error(), "reporter has")
	})

	// Test 5: Partial failures - some reports fail, others succeed
	s.Run("Partial failures", func() {
		// Create an invalid query data
		invalidQueryData := []byte("invalid")

		// Mock registry keeper for the successful query
		s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(spotSpec, nil).Once()

		rk.On("ReporterStake", s.ctx, addr2, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

		msg := &types.MsgBatchSubmitValue{
			Creator: addr2.String(),
			Values: []*types.SubmitValueItem{
				{
					QueryData: qDataBz,
					Value:     "000000000000000000000000000000000000000000000000000000000000001e",
				},
				{
					QueryData: invalidQueryData, // This should fail
					Value:     "000000000000000000000000000000000000000000000000000000000000002a",
				},
				{
					QueryData: []byte{}, // Empty query data should fail
					Value:     "000000000000000000000000000000000000000000000000000000000000002b",
				},
				{
					QueryData: qDataBz,
					Value:     "", // Empty value should fail
				},
			},
		}

		res, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.NoError(err) // Batch should succeed even with partial failures
		require.NotNil(res)
		require.Len(res.FailedIndices, 3) // Indices 1, 2, and 3 should fail
		require.Contains(res.FailedIndices, uint32(1))
		require.Contains(res.FailedIndices, uint32(2))
		require.Contains(res.FailedIndices, uint32(3))
	})

	// Test 6: ReporterStake error
	s.Run("ReporterStake error", func() {
		rk.On("ReporterStake", s.ctx, addr, queryId).Return(math.ZeroInt(), errors.New("reporter error")).Once()

		msg := &types.MsgBatchSubmitValue{
			Creator: addr.String(),
			Values: []*types.SubmitValueItem{
				{
					QueryData: qDataBz,
					Value:     "000000000000000000000000000000000000000000000000000000000000001e",
				},
			},
		}

		_, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.Error(err)
		require.Contains(err.Error(), "reporter error")
	})

	// Test 7: Token bridge deposit handling
	s.Run("Token bridge deposit", func() {
		// Setup bridge deposit query data
		bridgeQueryData := "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b54524242726964676556320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000003e8"
		bridgeQueryDataBz, err := hex.DecodeString(bridgeQueryData)
		require.NoError(err)
		bridgeQueryId := utils.QueryIDFromData(bridgeQueryDataBz)

		rk.On("ReporterStake", s.ctx, addr, bridgeQueryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

		// Mock registry keeper for bridge spec
		s.registryKeeper.On("GetSpec", s.ctx, "TRBBridgeV2").Return(bridgeSpec, nil).Once()
		s.bridgeKeeper.On("GetDepositStatus", s.ctx, uint64(1000)).Return(false, nil).Once()

		msg := &types.MsgBatchSubmitValue{
			Creator: addr.String(),
			Values: []*types.SubmitValueItem{
				{
					QueryData: bridgeQueryDataBz,
					Value:     bridgeDepositValue,
				},
			},
		}

		res, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
		require.NoError(err)
		require.NotNil(res)
		require.Empty(res.FailedIndices)
	})
}

func (s *KeeperTestSuite) TestBatchSubmitValue_EmptyBatch() {
	require := s.Require()
	addr := sample.AccAddressBytes()

	msg := &types.MsgBatchSubmitValue{
		Creator: addr.String(),
		Values:  []*types.SubmitValueItem{}, // Empty batch
	}

	_, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
	require.Error(err)
	require.Contains(err.Error(), "no values in batch")
}

func (s *KeeperTestSuite) TestBatchSubmitValue_AllFailures() {
	require := s.Require()
	addr := sample.AccAddressBytes()

	msg := &types.MsgBatchSubmitValue{
		Creator: addr.String(),
		Values: []*types.SubmitValueItem{
			{
				QueryData: []byte{}, // Invalid - empty
				Value:     "000000000000000000000000000000000000000000000000000000000000001e",
			},
			{
				QueryData: []byte("invalid"), // Invalid - not a valid query
				Value:     "",                // Invalid - empty value
			},
			{
				QueryData: nil, // Invalid - nil
				Value:     "test",
			},
		},
	}

	_, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
	require.Error(err)
	require.Contains(err.Error(), "all reports in batch failed")
}

func (s *KeeperTestSuite) TestBatchSubmitValue_StakeUsesFirstViableReport() {
	require := s.Require()
	k := s.oracleKeeper
	rk := s.reporterKeeper
	addr := sample.AccAddressBytes()

	qDataBz, err := utils.QueryBytesFromString(qData)
	require.NoError(err)
	queryId := utils.QueryIDFromData(qDataBz)

	query := types.QueryMeta{
		Id:                      1,
		Amount:                  math.NewInt(100_000),
		Expiration:              20,
		RegistrySpecBlockWindow: 10,
		HasRevealedReports:      false,
		QueryData:               qDataBz,
		QueryType:               "SpotPrice",
		CycleList:               true,
	}
	require.NoError(k.Query.Set(s.ctx, collections.Join(queryId, query.Id), query))
	require.NoError(k.QueryDataLimit.Set(s.ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()}))

	params, err := k.Params.Get(s.ctx)
	require.NoError(err)
	minStakeAmt := params.MinStakeAmount

	s.registryKeeper.On("GetSpec", s.ctx, "SpotPrice").Return(spotSpec, nil).Once()
	// ReporterStake must use the first viable report's query id (index 2), not the invalid entries before it.
	rk.On("ReporterStake", s.ctx, addr, queryId).Return(minStakeAmt.Add(math.NewInt(100)), nil).Once()

	msg := &types.MsgBatchSubmitValue{
		Creator: addr.String(),
		Values: []*types.SubmitValueItem{
			{
				QueryData: []byte{},
				Value:     "000000000000000000000000000000000000000000000000000000000000001e",
			},
			{
				QueryData: []byte("invalid"),
				Value:     "",
			},
			{
				QueryData: qDataBz,
				Value:     "000000000000000000000000000000000000000000000000000000000000001e",
			},
		},
	}

	res, err := s.msgServer.BatchSubmitValue(s.ctx, msg)
	require.NoError(err)
	require.NotNil(res)
	require.Equal([]uint32{0, 1}, res.FailedIndices)
}
