package keeper_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
)

func (s *KeeperTestSuite) TestPreventBridgeWithdrawalReport() {
	require := s.Require()
	k := s.oracleKeeper
	queryTypeString := "TRBBridgeV2"
	toLayerBool := true
	withdrawalId := uint64(1)
	withdrawalIdUint64 := new(big.Int).SetUint64(withdrawalId)
	err := k.QueryDataLimit.Set(s.ctx, types.QueryDataLimit{Limit: types.InitialQueryDataLimit()})
	require.NoError(err)
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

	s.bridgeKeeper.On("GetDepositStatus", s.ctx, uint64(1)).Return(false, nil)

	_, err = k.PreventBridgeWithdrawalReport(s.ctx, queryDataEncoded)
	require.NoError(err)

	// try with toLayerBool false
	toLayerBool = false
	queryDataArgsEncoded, err = queryDataArgs.Pack(toLayerBool, withdrawalIdUint64)
	require.NoError(err)
	queryDataEncoded, err = finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	require.NoError(err)
	_, err = k.PreventBridgeWithdrawalReport(s.ctx, queryDataEncoded)
	require.Error(err)

	// try with trb/usd
	queryBytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	_, err = k.PreventBridgeWithdrawalReport(s.ctx, queryBytes)
	require.NoError(err)
}
