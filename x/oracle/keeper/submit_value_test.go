package keeper_test

import (
	"errors"
	"time"

	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/utils"
	"github.com/tellor-io/layer/x/oracle/types"
	oracleutils "github.com/tellor-io/layer/x/oracle/utils"
	regtypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
)

func (s *KeeperTestSuite) TestSetValue() {
	require := s.Require()
	ctx := s.ctx
	k := s.oracleKeeper
	regK := s.registryKeeper

	reporter := sample.AccAddressBytes()
	queryId, err := utils.QueryIDFromDataString(queryData)
	require.NoError(err)
	querydatabytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	query := types.QueryMeta{
		Id:                    1,
		Amount:                math.NewInt(1_000_000),
		Expiration:            ctx.BlockTime().Add(time.Hour),
		RegistrySpecTimeframe: time.Hour,
		HasRevealedReports:    false,
		QueryData:             querydatabytes,
		QueryType:             "SpotPrice",
	}

	queryBytes, err := utils.QueryBytesFromString(queryData)
	require.NoError(err)
	regK.On("GetSpec", ctx, "SpotPrice").Return(regtypes.GenesisDataSpec(), nil)
	err = k.SetValue(ctx, reporter, query, "0x0000000000000000000000000000000000000000000000000000000000000009", queryBytes, 1, true)
	require.NoError(err)

	report, err := k.Reports.Get(ctx, collections.Join3(queryId, reporter.Bytes(), uint64(1)))
	require.NoError(err)
	require.Equal(report.Value, "0x0000000000000000000000000000000000000000000000000000000000000009")
	require.Equal(report.QueryId, queryId)
	require.Equal(report.Reporter, reporter.String())
	require.Equal(report.QueryType, "SpotPrice")
	require.Equal(report.Power, uint64(1))
	require.Equal(report.BlockNumber, uint64(ctx.BlockHeight()))
}

func (s *KeeperTestSuite) TestVerifyCommit() {
	require := s.Require()
	ctx := s.ctx
	// k := s.oracleKeeper

	// good hash
	reporter := sample.AccAddress()
	salt, err := oracleutils.Salt(32)
	require.NoError(err)
	value := "0x0000000000000000000000000000000000000000000000000000000000000009"
	hash := oracleutils.CalculateCommitment(value, salt)
	res := s.VerifyCommit(ctx, reporter, value, salt, hash)
	require.True(res)

	// bad hash
	res = s.VerifyCommit(ctx, reporter, value, salt, "0x0000000000000000000000000000000000000000000000000000000000000000")
	require.False(res)

	// empty hash
	res = s.VerifyCommit(ctx, reporter, value, salt, "")
	require.False(res)

	// bad value
	res = s.VerifyCommit(ctx, reporter, "0x0000000000000000000000000000000000000000000000000000000000000000", salt, hash)
	require.False(res)

	// empty value
	res = s.VerifyCommit(ctx, reporter, "", salt, hash)
	require.False(res)

	// bad salt
	res = s.VerifyCommit(ctx, reporter, value, "0x0000000000000000000000000000000000000000000000000000000000000000", hash)
	require.False(res)

	// empty salt
	res = s.VerifyCommit(ctx, reporter, value, "", hash)
	require.False(res)

	// empty entries
	res = s.VerifyCommit(ctx, "", "", "", "")
	require.False(res)
}

func (s *KeeperTestSuite) TestGetDataSpec() {
	require := s.Require()
	ctx := s.ctx
	k := s.oracleKeeper
	regK := s.registryKeeper

	expectedABI := []*regtypes.ABIComponent{
		{Name: "asset", FieldType: "string"},
		{Name: "currency", FieldType: "string"},
	}
	regK.On("GetSpec", ctx, "SpotPrice").Return(regtypes.GenesisDataSpec(), nil).Once()
	spec, err := k.GetDataSpec(ctx, "SpotPrice")
	require.NoError(err)
	require.Equal(spec.AbiComponents, expectedABI)
	require.Equal(spec.DocumentHash, "")
	require.Equal(spec.AggregationMethod, "weighted-median")
	require.Equal(spec.Registrar, "genesis")
	require.Equal(spec.ReportBufferWindow, time.Second*10)
	require.Equal(spec.ResponseValueType, "uint256")

	regK.On("GetSpec", ctx, "BadQueryType").Return(regtypes.GenesisDataSpec(), errors.New("not found")).Once()
	spec, err = k.GetDataSpec(ctx, "BadQueryType")
	require.Error(err)
	require.Equal(spec, regtypes.DataSpec{})
}
