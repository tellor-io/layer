package keeper_test

import (
	"github.com/stretchr/testify/mock"
	"github.com/tellor-io/layer/testutil/sample"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (s *KeeperTestSuite) TestGetTimeBasedRewards() {
	require := s.Require()
	k := s.oracleKeeper
	ak := s.accountKeeper
	bk := s.bankKeeper
	ctx := s.ctx

	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount))
	bk.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}).Once()
	tbr := k.GetTimeBasedRewards(ctx)
	require.Equal(tbr, math.NewInt(100))

	bk.On("GetBalance", ctx, testModuleAccount.GetAddress(), "loya").Return(sdk.Coin{Amount: math.NewInt(0), Denom: "loya"}).Once()
	tbr = k.GetTimeBasedRewards(ctx)
	require.Equal(tbr, math.ZeroInt())
}

func (s *KeeperTestSuite) TestGetTimeBasedRewardsAccount() {
	require := s.Require()
	k := s.oracleKeeper
	ak := s.accountKeeper
	ctx := s.ctx

	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(nil)).Once()
	require.Equal(k.GetTimeBasedRewardsAccount(ctx), nil)

	add := sample.AccAddressBytes()
	baseAccount := authtypes.NewBaseAccountWithAddress(add)
	permissions := []string{authtypes.Minter, authtypes.Burner, authtypes.Staking}
	testModuleAccount := authtypes.NewModuleAccount(baseAccount, "time_based_rewards", permissions...)
	ak.On("GetModuleAccount", ctx, minttypes.TimeBasedRewards).Return(sdk.ModuleAccountI(testModuleAccount)).Once()
	require.Equal(k.GetTimeBasedRewardsAccount(ctx), testModuleAccount)
}

func (s *KeeperTestSuite) TestDistributeTipMissingReporterAddsDust() {
	require := s.Require()
	k := s.oracleKeeper
	ctx := s.ctx

	missing := sample.AccAddressBytes()
	present := sample.AccAddressBytes()
	queryId := []byte("tip-dust-qid")
	metaId := uint64(42)

	require.NoError(k.Reports.Set(ctx, collections.Join3(queryId, missing.Bytes(), metaId), types.MicroReport{
		Reporter: missing.String(),
		Power:    50,
		QueryId:  queryId,
		MetaId:   metaId,
	}))
	require.NoError(k.Reports.Set(ctx, collections.Join3(queryId, present.Bytes(), metaId), types.MicroReport{
		Reporter: present.String(),
		Power:    50,
		QueryId:  queryId,
		MetaId:   metaId,
	}))

	reward := math.LegacyNewDec(100)
	s.reporterKeeper.On("DivvyingTips", mock.Anything, missing, mock.Anything).
		Return(reportertypes.ErrReporterDoesNotExist).Once()
	s.reporterKeeper.On("DivvyingTips", mock.Anything, present, mock.Anything).
		Return(nil).Once()

	err := k.DistributeTip(ctx, types.Aggregate{
		MetaId:         metaId,
		QueryId:        queryId,
		AggregatePower: 100,
	}, reward)
	require.NoError(err)

	dust, err := k.Dust.Get(ctx)
	require.NoError(err)
	// missing reporter had 50/100 of the tip → 50 truncated into dust
	require.Equal(math.NewInt(50), dust)
}
