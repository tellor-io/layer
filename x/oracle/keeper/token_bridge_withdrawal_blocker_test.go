package keeper_test

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/tellor-io/layer/utils"
)

func (s *KeeperTestSuite) TestPreventBridgeWithdrawalReport() {
	require := s.Require()
	k := s.oracleKeeper

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
	_, err = k.PreventBridgeWithdrawalReport(queryDataEncoded)
	require.NoError(err)

	// try with toLayerBool false
	toLayerBool = false
	queryDataArgsEncoded, err = queryDataArgs.Pack(toLayerBool, withdrawalIdUint64)
	require.NoError(err)
	queryDataEncoded, err = finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	require.NoError(err)
	_, err = k.PreventBridgeWithdrawalReport(queryDataEncoded)
	require.Error(err)

	// try with trb/usd
	queryBytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	_, err = k.PreventBridgeWithdrawalReport(queryBytes)
	require.NoError(err)
}
