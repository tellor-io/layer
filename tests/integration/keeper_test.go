package integration_test

import (
	"time"

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
	"github.com/stretchr/testify/suite"

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
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"

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
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/oracle"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	"github.com/tellor-io/layer/x/registry"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/tests/integration"
)

const (
	ethQueryData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	btcQueryData = "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
)

type IntegrationTestSuite struct {
	suite.Suite

	oraclekeeper   oraclekeeper.Keeper
	disputekeeper  disputekeeper.Keeper
	registrykeeper registrykeeper.Keeper

	accountKeeper  authkeeper.AccountKeeper
	bankKeeper     bankkeeper.BaseKeeper
	slashingKeeper slashingkeeper.Keeper
	stakingKeeper  *stakingkeeper.Keeper
	ctx            sdk.Context
	appCodec       codec.Codec
	authConfig     *authmodulev1.Module

	queryHelper       *baseapp.QueryServiceTestHelper
	interfaceRegistry codectypes.InterfaceRegistry
	fetchStoreKey     func(string) storetypes.StoreKey
}

func (suite *IntegrationTestSuite) initKeepersWithmAccPerms(blockedAddrs map[string]bool) (authkeeper.AccountKeeper, bankkeeper.BaseKeeper) {
	maccPerms := map[string][]string{}
	for _, permission := range suite.authConfig.ModuleAccountPermissions {
		maccPerms[permission.Account] = permission.Permissions
	}

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	cdc := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Amino

	maccPerms[authtypes.Burner] = []string{authtypes.Burner}
	maccPerms[authtypes.Minter] = []string{authtypes.Minter}

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

	suite.registrykeeper = *registrykeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(registrytypes.StoreKey), suite.fetchStoreKey(registrytypes.StoreKey), paramtypes.Subspace{},
	)

	suite.oraclekeeper = *oraclekeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(oracletypes.StoreKey), suite.fetchStoreKey(oracletypes.StoreKey), paramtypes.Subspace{}, authKeeper, bankKeeper, stakingKeeper, suite.registrykeeper,
	)

	suite.disputekeeper = *disputekeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(disputetypes.StoreKey), suite.fetchStoreKey(disputetypes.StoreKey), paramtypes.Subspace{}, authKeeper, bankKeeper, slashingKeeper, stakingKeeper,
	)

	return authKeeper, bankKeeper
}

func (suite *IntegrationTestSuite) SetupTest() {
	registry.AppWiringSetup()
	dispute.AppWiringSetup()
	oracle.AppWiringSetup()
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
			integration.AuthModule(),
			configurator.BankModule(),
			configurator.StakingModule(),
			configurator.SlashingModule(),
			configurator.ParamsModule(),
			configurator.ConsensusModule(),
			integration.OracleModule(),
			integration.DisputeModule(),
			integration.RegistryModule()),
		&suite.accountKeeper, &suite.bankKeeper, &suite.stakingKeeper, &suite.slashingKeeper,
		&suite.interfaceRegistry, &suite.appCodec, &suite.authConfig, &suite.oraclekeeper,
		&suite.disputekeeper, &suite.registrykeeper)

	suite.NoError(err)
	suite.ctx = app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})
	suite.fetchStoreKey = app.UnsafeFindStoreKey

	suite.queryHelper = baseapp.NewQueryServerTestHelper(suite.ctx, suite.interfaceRegistry)
}

func (suite *IntegrationTestSuite) mintTokens(addr sdk.AccAddress) {
	ctx := suite.ctx
	require := suite.Require()
	suite.accountKeeper.SetAccount(ctx, authtypes.NewBaseAccountWithAddress(addr))
	_, bank := suite.initKeepersWithmAccPerms(make(map[string]bool))
	require.NoError(bank.MintCoins(ctx, authtypes.Minter, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(1000000)))))
	require.NoError(bank.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(100000)))))
}

func (suite *IntegrationTestSuite) newKeysWithTokens() (sdk.AccAddress, string) {
	denom := sdk.DefaultBondDenom
	PrivKey := secp256k1.GenPrivKey()
	PubKey := PrivKey.PubKey()
	Addr := sdk.AccAddress(PubKey.Address())
	suite.mintTokens(Addr)
	return Addr, denom
}

func (suite *IntegrationTestSuite) microReport() (disputetypes.MicroReport, sdk.ValAddress) {
	val := suite.stakingKeeper.GetAllValidators(suite.ctx)[0]
	valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
	suite.Require().NoError(err)
	return disputetypes.MicroReport{
		Reporter:  sdk.AccAddress(valAddr).String(),
		Power:     val.GetConsensusPower(val.GetBondedTokens()),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}, valAddr

}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
