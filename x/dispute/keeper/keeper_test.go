package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/mocks"
	"github.com/tellor-io/layer/x/dispute/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	disputeKeeper keeper.Keeper

	accountKeeper  *mocks.AccountKeeper
	bankKeeper     *mocks.BankKeeper
	oracleKeeper   *mocks.OracleKeeper
	reporterKeeper *mocks.ReporterKeeper

	msgServer types.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	config.SetupConfig()

	s.disputeKeeper,
		s.oracleKeeper,
		s.reporterKeeper,
		s.accountKeeper,
		s.bankKeeper,
		s.ctx = keepertest.DisputeKeeper(s.T())

	s.msgServer = keeper.NewMsgServerImpl(s.disputeKeeper)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestLogger() {
	require := s.Require()
	logger := s.disputeKeeper.Logger(s.ctx)
	require.NotNil(logger)
}
