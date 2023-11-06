package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil/configurator"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tellor-io/layer/x/dispute/keeper"

	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	"cosmossdk.io/core/appconfig"
	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	_ "github.com/cosmos/cosmos-sdk/x/bank"
	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/mint"
	_ "github.com/cosmos/cosmos-sdk/x/params"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
	_ "github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/tellor-io/layer/x/dispute"
	"github.com/tellor-io/layer/x/dispute/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	modulev1 "github.com/tellor-io/layer/api/layer/dispute/module"
	"github.com/tellor-io/layer/app"
)

const (
	holder     = "holder"
	multiPerm  = "multiple permissions account"
	randomPerm = "random permission"
)

var (
	burnerAcc = authtypes.NewEmptyModuleAccount(authtypes.Burner, authtypes.Burner)
)

type IntegrationTestSuite struct {
	suite.Suite

	disputekeeper  keeper.Keeper
	accountKeeper  authkeeper.AccountKeeper
	bankKeeper     bankkeeper.BaseKeeper
	slashingKeeper slashingkeeper.Keeper
	stakingKeeper  *stakingkeeper.Keeper
	ctx            sdk.Context
	appCodec       codec.Codec
	authConfig     *authmodulev1.Module

	queryClient   types.QueryClient
	msgServer     types.MsgServer
	fetchStoreKey func(string) storetypes.StoreKey
}

func (suite *IntegrationTestSuite) initKeepersWithmAccPerms(blockedAddrs map[string]bool) (authkeeper.AccountKeeper, bankkeeper.BaseKeeper) {
	maccPerms := map[string][]string{}
	for _, permission := range suite.authConfig.ModuleAccountPermissions {
		maccPerms[permission.Account] = permission.Permissions
	}

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	cdc := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Amino

	maccPerms[holder] = nil
	maccPerms[authtypes.Burner] = []string{authtypes.Burner}
	maccPerms[authtypes.Minter] = []string{authtypes.Minter}
	maccPerms[multiPerm] = []string{authtypes.Burner, authtypes.Minter, authtypes.Staking}
	maccPerms[randomPerm] = []string{"random"}
	authKeeper := authkeeper.NewAccountKeeper(
		appCodec, suite.fetchStoreKey(banktypes.StoreKey), authtypes.ProtoBaseAccount,
		maccPerms, sdk.Bech32MainPrefix, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	bankKeeper := bankkeeper.NewBaseKeeper(
		appCodec, suite.fetchStoreKey(banktypes.StoreKey), authKeeper, blockedAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(banktypes.StoreKey), authKeeper, bankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	slashingKeeper := slashingkeeper.NewKeeper(
		appCodec, cdc, suite.fetchStoreKey(banktypes.StoreKey), stakingKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	disputeKeeper := keeper.NewKeeper(
		appCodec, suite.fetchStoreKey(types.StoreKey), suite.fetchStoreKey(types.StoreKey), paramtypes.Subspace{}, authKeeper, bankKeeper, slashingKeeper, stakingKeeper,
	)
	suite.disputekeeper = *disputeKeeper

	return authKeeper, bankKeeper
}

func AuthModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.ModuleConfigs["auth"] = &appv1alpha1.ModuleConfig{
			Name: "auth",
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix: "cosmos",
				ModuleAccountPermissions: []*authmodulev1.ModuleAccountPermission{
					{Account: "fee_collector"},
					{Account: "distribution"},
					{Account: "dispute"},
					{Account: "mint", Permissions: []string{"minter"}},
					{Account: "bonded_tokens_pool", Permissions: []string{"burner", "staking"}},
					{Account: "not_bonded_tokens_pool", Permissions: []string{"burner", "staking"}},
					{Account: "gov", Permissions: []string{"burner"}},
					{Account: "nft"},
				},
			}),
		}
	}
}

func DisputeModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.BeginBlockersOrder = append(config.BeginBlockersOrder, "dispute")
		config.EndBlockersOrder = append(config.EndBlockersOrder, "dispute")
		config.InitGenesisOrder = append(config.InitGenesisOrder, "dispute")
		config.ModuleConfigs["dispute"] = &appv1alpha1.ModuleConfig{
			Name:   "dispute",
			Config: appconfig.WrapAny(&modulev1.Module{}),
		}
	}
}

func (suite *IntegrationTestSuite) SetupTest() {
	dispute.AppWiringSetup()
	var interfaceRegistry codectypes.InterfaceRegistry
	accountPubKeyPrefix := app.AccountAddressPrefix + "pub"
	validatorAddressPrefix := app.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := app.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := app.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := app.AccountAddressPrefix + "valconspub"
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(app.AccountAddressPrefix, accountPubKeyPrefix)
	config.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)

	app, err := sims.Setup(
		configurator.NewAppConfig(
			AuthModule(),
			configurator.BankModule(),
			configurator.StakingModule(),
			configurator.SlashingModule(),
			configurator.ParamsModule(),
			configurator.ConsensusModule(),
			DisputeModule()),
		&suite.accountKeeper, &suite.bankKeeper, &suite.stakingKeeper, &suite.slashingKeeper,
		&interfaceRegistry, &suite.appCodec, &suite.authConfig, &suite.disputekeeper)

	suite.NoError(err)
	suite.ctx = app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})
	suite.fetchStoreKey = app.UnsafeFindStoreKey

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, interfaceRegistry)
	types.RegisterQueryServer(queryHelper, suite.disputekeeper)
	queryClient := types.NewQueryClient(queryHelper)
	types.RegisterInterfaces(interfaceRegistry)

	suite.queryClient = queryClient
	suite.msgServer = keeper.NewMsgServerImpl(suite.disputekeeper)

}

func (suite *IntegrationTestSuite) TestVotingOnDispute() {
	ctx := suite.ctx
	denom := sdk.DefaultBondDenom
	require := suite.Require()

	PrivKey := secp256k1.GenPrivKey()
	PubKey := PrivKey.PubKey()
	Addr := sdk.AccAddress(PubKey.Address())
	_, keeper := suite.initKeepersWithmAccPerms(make(map[string]bool))
	require.NoError(keeper.MintCoins(ctx, authtypes.Minter, sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(100000000000000)))))
	require.NoError(keeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, Addr, sdk.NewCoins(sdk.NewCoin(denom, sdk.NewInt(100000000)))))

	suite.initKeepersWithmAccPerms(make(map[string]bool))
	report, valAddr := suite.microReport()
	// Propose dispute pay half of the fee from account
	_, err := suite.msgServer.ProposeDispute(ctx, &types.MsgProposeDispute{
		Creator:         Addr.String(),
		Report:          &report,
		Fee:             sdk.NewCoin(denom, sdk.NewInt(5000)),
		DisputeCategory: types.Warning,
	})
	require.Equal(uint64(1), suite.disputekeeper.GetDisputeCount(ctx))
	require.Equal(1, len(suite.disputekeeper.GetOpenDisputeIds(ctx).Ids))
	require.NoError(err)
	// check validator wasn't slashed/jailed
	val, found := suite.stakingKeeper.GetValidator(ctx, valAddr)
	bondedTokensBefore := val.GetBondedTokens()
	require.True(found)
	require.False(val.IsJailed())
	require.Equal(bondedTokensBefore, sdk.NewInt(1000000))
	// Add dispute fee to complete the fee and jail/slash validator
	_, err = suite.msgServer.AddFeeToDispute(ctx, &types.MsgAddFeeToDispute{
		Creator:   Addr.String(),
		DisputeId: 0,
		Amount:    sdk.NewCoin(denom, sdk.NewInt(5000)),
	})
	require.NoError(err)
	// check validator was slashed/jailed
	val, found = suite.stakingKeeper.GetValidator(ctx, valAddr)
	require.True(found)
	require.True(val.IsJailed())
	// check validator was slashed 1% of tokens
	require.Equal(val.GetBondedTokens(), bondedTokensBefore.Sub(bondedTokensBefore.Mul(math.NewInt(1)).Quo(math.NewInt(100))))

	ids := suite.disputekeeper.CheckPrevoteDisputesForExpiration(ctx)
	suite.disputekeeper.StartVoting(ctx, ids)

}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (suite *IntegrationTestSuite) microReport() (types.MicroReport, sdk.ValAddress) {
	val := suite.stakingKeeper.GetAllValidators(suite.ctx)[0]
	valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
	suite.Require().NoError(err)
	return types.MicroReport{
		Reporter:  sdk.AccAddress(valAddr).String(),
		Power:     val.GetConsensusPower(val.GetBondedTokens()),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}, valAddr

}
