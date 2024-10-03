package keeper_test

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"

	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestGetTokenBridgeDeposit() {
	require := s.Require()
	k := s.oracleKeeper
	// regK := s.registryKeeper
	ctx := s.ctx

	// try trb/usd spot price, should err with NotTokenDeposit
	queryBytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	res, err := k.TokenBridgeDepositCheck(ctx, queryBytes)
	require.ErrorContains(err, types.ErrNotTokenDeposit.Error())
	require.Equal(types.QueryMeta{}, res)

	// build TRBBridge queryData with method from x/bridge/withdraw_tokens.go
	queryTypeString := "TRBBridge"
	toLayerBool := true
	withdrawalId := uint64(1)
	withdrawalIdUint64 := new(big.Int).SetUint64(withdrawalId)
	// prepare encoding
	StringType, err := abi.NewType("string", "", nil)
	require.NoError(err)
	Uint256Type, err := abi.NewType("uint256", "", nil)
	require.NoError(err)
	BoolType, err := abi.NewType("bool", "", nil)
	require.NoError(err)
	BytesType, err := abi.NewType("bytes", "", nil)
	require.NoError(err)
	// encode query data arguments first
	queryDataArgs := abi.Arguments{
		{Type: BoolType},
		{Type: Uint256Type},
	}
	queryDataArgsEncoded, err := queryDataArgs.Pack(toLayerBool, withdrawalIdUint64)
	require.NoError(err)
	// encode query data
	finalArgs := abi.Arguments{
		{Type: StringType},
		{Type: BytesType},
	}
	queryDataEncoded, err := finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	require.NoError(err)

	res, err = k.TokenBridgeDepositCheck(ctx, queryDataEncoded)
	require.NoError(err)
	require.Equal(res.QueryType, "TRBBridge")
	require.Equal(res.Amount, math.NewInt(0))
	require.Equal(res.Expiration, ctx.BlockTime().Add(time.Hour))
	require.Equal(res.RegistrySpecBlockWindow, time.Hour)

	// try TRBBridge but toLayer is false
	toLayerBool = false
	withdrawalIdUint64 = new(big.Int).SetUint64(2)
	queryDataArgsEncoded, err = queryDataArgs.Pack(toLayerBool, withdrawalIdUint64)
	require.NoError(err)
	queryDataEncoded, err = finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	require.NoError(err)
	res, err = k.TokenBridgeDepositCheck(ctx, queryDataEncoded)
	require.ErrorContains(err, types.ErrNotTokenDeposit.Error())
	require.Equal(types.QueryMeta{}, res)
}

// func (s *KeeperTestSuite) TestHandleBridgeDepositCommit() {
// 	require := s.Require()
// 	k := s.oracleKeeper
// 	ctx := s.ctx
// 	ctx = ctx.WithBlockHeight(10)
// 	queryId, _ := utils.QueryIDFromDataString("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0")

// 	queryMeta := types.QueryMeta{
// 		Id:                    1,
// 		Amount:                math.NewInt(100 * 1e6),
// 		Expiration:            10 + 10,
// 		RegistrySpecBlockWindow: 10,
// 		HasRevealedReports:    false,
// 		QueryData:             []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
// 		QueryType:             "TRBBridge",
// 	}

// 	testCases := []struct {
// 		name      string
// 		setup     func()
// 		queryMeta types.QueryMeta
// 		queryId   []byte
// 		err       bool
// 		checks    func()
// 	}{
// 		{
// 			name:      "tipped and window not expired",
// 			queryMeta: queryMeta,
// 			err:       false,
// 		},
// 		{
// 			name: "tipped and window expired before offset",
// 			queryMeta: types.QueryMeta{
// 				Id:                    2,
// 				Amount:                math.NewInt(100 * 1e6),
// 				Expiration:            5,
// 				RegistrySpecBlockWindow: 10,
// 				HasRevealedReports:    false,
// 				QueryData:             []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
// 				QueryType:             "TRBBridge",
// 			},
// 			err: false,
// 			checks: func() {
// 				query, err := k.Query.Get(ctx, collections.Join(queryId, uint64(2)))
// 				require.NoError(err)
// 				require.Equal(query.Expiration, ctx.BlockHeight() + int64(queryMeta.RegistrySpecBlockWindow))
// 			},
// 		},
// 		{
// 			name: "no tip and expired before blocktime",
// 			queryMeta: types.QueryMeta{
// 				Id:                    3,
// 				Amount:                math.NewInt(0),
// 				Expiration:            5,
// 				RegistrySpecBlockWindow: 2,
// 				HasRevealedReports:    false,
// 				QueryData:             []byte("0x5c13cd9c97dbb98f2429c101a2a8150e6c7a0ddaff6124ee176a3a411067ded0"),
// 				QueryType:             "TRBBridge",
// 			},
// 			err: false,
// 			checks: func() {
// 				query, err := k.Query.Get(ctx, collections.Join(queryId, uint64(0)))
// 				require.NoError(err)
// 				require.Equal(query.Expiration, uint64(ctx.BlockHeight()) + queryMeta.RegistrySpecBlockWindow)
// 				require.Equal(query.Id, uint64(0))
// 			},
// 		},
// 	}
// 	for _, tc := range testCases {
// 		s.Run(tc.name, func() {
// 			fmt.Println("TEST: ", tc.name)
// 			if tc.setup != nil {
// 				tc.setup()
// 			}
// 			reporterAcc := sample.AccAddressBytes()
// 			err := k.HandleBridgeDepositDirectReveal(ctx, queryId, tc.queryMeta, reporterAcc, "hash")
// 			if tc.err {
// 				require.Error(err)
// 			} else {
// 				require.NoError(err)
// 			}
// 			if tc.checks != nil {
// 				tc.checks()
// 			}
// 		})
// 	}
// }
