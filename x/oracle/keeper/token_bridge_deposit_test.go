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
	res, err := k.TokenBridgeDepositCheck(ctx.BlockTime(), queryBytes)
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

	res, err = k.TokenBridgeDepositCheck(ctx.BlockTime(), queryDataEncoded)
	require.NoError(err)
	require.Equal(res.QueryType, "TRBBridge")
	require.Equal(res.Amount, math.NewInt(0))
	require.Equal(res.Expiration, ctx.BlockTime().Add(time.Hour))
	require.Equal(res.RegistrySpecTimeframe, time.Second)

	// try TRBBridge but toLayer is false
	toLayerBool = false
	withdrawalIdUint64 = new(big.Int).SetUint64(2)
	queryDataArgsEncoded, err = queryDataArgs.Pack(toLayerBool, withdrawalIdUint64)
	require.NoError(err)
	queryDataEncoded, err = finalArgs.Pack(queryTypeString, queryDataArgsEncoded)
	require.NoError(err)
	res, err = k.TokenBridgeDepositCheck(ctx.BlockTime(), queryDataEncoded)
	require.ErrorContains(err, types.ErrNotTokenDeposit.Error())
	require.Equal(types.QueryMeta{}, res)
}