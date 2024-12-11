package oracle_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/oracle"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestSuite struct {
	suite.Suite

	ctx          sdk.Context
	oracleKeeper keeper.Keeper

	reporterKeeper *mocks.ReporterKeeper
	registryKeeper *mocks.RegistryKeeper
	accountKeeper  *mocks.AccountKeeper
	bankKeeper     *mocks.BankKeeper
	bridgeKeeper   *mocks.BridgeKeeper
}

func (s *TestSuite) SetupTest() {
	config.SetupConfig()

	s.oracleKeeper,
		s.reporterKeeper,
		s.registryKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.bridgeKeeper,
		s.ctx = keepertest.OracleKeeper(s.T())
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestEndBlocker() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	query1, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.NotNil(query1)

	s.registryKeeper.On("GetSpec", ctx, "SpotPrice").Return(registrytypes.DataSpec{}, nil)
	err = oracle.EndBlocker(ctx, k)
	require.NoError(err)

	query2, err := k.GetCurrentQueryInCycleList(ctx)
	require.NoError(err)
	require.NotNil(query2)
	require.NotEqual(query1, query2)
}
