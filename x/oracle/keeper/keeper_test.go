package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/mocks"
	"github.com/tellor-io/layer/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	value = "000000000000000000000000000000000000000000000058528649cf80ee0000"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	oracleKeeper   keeper.Keeper
	bankKeeper     *mocks.BankKeeper
	accountKeeper  *mocks.AccountKeeper
	registryKeeper *mocks.RegistryKeeper
	reporterKeeper *mocks.ReporterKeeper

	queryClient types.QueryServer
	msgServer   types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	config.SetupConfig()

	s.oracleKeeper,
		s.reporterKeeper,
		s.registryKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.ctx = keepertest.OracleKeeper(s.T())

	s.msgServer = keeper.NewMsgServerImpl(s.oracleKeeper)
	s.queryClient = keeper.NewQuerier(s.oracleKeeper)

	// Initialize params
	s.NoError(s.oracleKeeper.SetParams(s.ctx, types.DefaultParams()))
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
