package setup

import (
	"strconv"
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
	disputemodulev1 "github.com/tellor-io/layer/api/layer/dispute/module"
	mintmodulev1 "github.com/tellor-io/layer/api/layer/mint/module"
	oraclemodulev1 "github.com/tellor-io/layer/api/layer/oracle/module"
	registrymodulev1 "github.com/tellor-io/layer/api/layer/registry/module"
	reportermodulev1 "github.com/tellor-io/layer/api/layer/reporter/module"
	"github.com/tellor-io/layer/app/config"
	_ "github.com/tellor-io/layer/x/dispute"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	_ "github.com/tellor-io/layer/x/mint"
	mintkeeper "github.com/tellor-io/layer/x/mint/keeper"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	_ "github.com/tellor-io/layer/x/oracle"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	_ "github.com/tellor-io/layer/x/registry/module"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	_ "github.com/tellor-io/layer/x/reporter/module"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	"cosmossdk.io/core/appconfig"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/configurator"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	_ "github.com/cosmos/cosmos-sdk/x/mint"
	_ "github.com/cosmos/cosmos-sdk/x/params"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func AuthModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.ModuleConfigs["auth"] = &appv1alpha1.ModuleConfig{
			Name: "auth",
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix: "tellor",
				ModuleAccountPermissions: []*authmodulev1.ModuleAccountPermission{
					{Account: "fee_collector"},
					{Account: "distribution"},
					{Account: "oracle", Permissions: []string{"burner"}},
					{Account: "dispute", Permissions: []string{"burner"}},
					{Account: "registry"},
					{Account: "mint", Permissions: []string{"minter"}},
					{Account: "time_based_rewards"},
					{Account: "mint_to_team"},
					{Account: "bonded_tokens_pool", Permissions: []string{"burner", "staking"}},
					{Account: "not_bonded_tokens_pool", Permissions: []string{"burner", "staking"}},
					{Account: "gov", Permissions: []string{"burner"}},
					{Account: "nft"},
					{Account: "reporter"},
					{Account: "tips_escrow_pool"},
				},
			}),
		}
	}
}

func OracleModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.BeginBlockersOrder = append(config.BeginBlockersOrder, "oracle")
		config.EndBlockersOrder = append(config.EndBlockersOrder, "oracle")
		config.InitGenesisOrder = append(config.InitGenesisOrder, "oracle")
		config.ModuleConfigs["oracle"] = &appv1alpha1.ModuleConfig{
			Name:   "oracle",
			Config: appconfig.WrapAny(&oraclemodulev1.Module{}),
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
			Config: appconfig.WrapAny(&disputemodulev1.Module{}),
		}
	}
}

func RegistryModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.BeginBlockersOrder = append(config.BeginBlockersOrder, "registry")
		config.EndBlockersOrder = append(config.EndBlockersOrder, "registry")
		config.InitGenesisOrder = append(config.InitGenesisOrder, "registry")
		config.ModuleConfigs["registry"] = &appv1alpha1.ModuleConfig{
			Name:   "registry",
			Config: appconfig.WrapAny(&registrymodulev1.Module{}),
		}
	}
}

func MintModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.ModuleConfigs["mint"] = &appv1alpha1.ModuleConfig{
			Name:   "mint",
			Config: appconfig.WrapAny(&mintmodulev1.Module{}),
		}
	}
}

func ReporterModule() configurator.ModuleOption {
	return func(config *configurator.Config) {
		config.BeginBlockersOrder = append(config.BeginBlockersOrder, "reporter")
		config.EndBlockersOrder = append(config.EndBlockersOrder, "reporter")
		config.InitGenesisOrder = append(config.InitGenesisOrder, "reporter")
		config.ModuleConfigs["reporter"] = &appv1alpha1.ModuleConfig{
			Name:   "reporter",
			Config: appconfig.WrapAny(&reportermodulev1.Module{}),
		}
	}
}

// func BridgeModule() configurator.ModuleOption {
// 	return func(config *configurator.Config) {
// 		config.BeginBlockersOrder = append(config.BeginBlockersOrder, "bridge")
// 		config.EndBlockersOrder = append(config.EndBlockersOrder, "bridge")
// 		config.InitGenesisOrder = append(config.InitGenesisOrder, "bridge")
// 		config.ModuleConfigs["bridge"] = &appv1alpha1.ModuleConfig{
// 			Name:   "bridge",
// 			Config: appconfig.WrapAny(&bridgemodulev1.Module{}),
// 		}
// 	}
// }

func DefaultStartUpConfig() simtestutil.StartupConfig {
	priv := secp256k1.GenPrivKey()
	ba := authtypes.NewBaseAccount(priv.PubKey().Address().Bytes(), priv.PubKey(), 0, 0)
	ga := simtestutil.GenesisAccount{GenesisAccount: ba, Coins: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()))}
	return simtestutil.StartupConfig{
		ValidatorSet:    simtestutil.CreateRandomValidatorSet,
		AtGenesis:       false,
		GenesisAccounts: []simtestutil.GenesisAccount{ga},
		DB:              dbm.NewMemDB(),
	}
}

type SharedSetup struct {
	Oraclekeeper   oraclekeeper.Keeper
	Disputekeeper  disputekeeper.Keeper
	Registrykeeper registrykeeper.Keeper
	Mintkeeper     mintkeeper.Keeper
	Reporterkeeper reporterkeeper.Keeper

	Accountkeeper  authkeeper.AccountKeeper
	Bankkeeper     bankkeeper.BaseKeeper
	distrKeeper    distrkeeper.Keeper
	slashingKeeper slashingkeeper.Keeper
	Stakingkeeper  *stakingkeeper.Keeper
	Govkeeper      *govkeeper.Keeper
	Ctx            sdk.Context
	appCodec       codec.Codec
	authConfig     *authmodulev1.Module

	queryHelper       *baseapp.QueryServiceTestHelper
	interfaceRegistry codectypes.InterfaceRegistry
	fetchStoreKey     func(string) storetypes.StoreKey

	Denom   string
	App     *runtime.App
	require *require.Assertions
}

func (s *SharedSetup) initKeepersWithmAccPerms(blockedAddrs map[string]bool) {
	maccPerms := map[string][]string{}
	for _, permission := range s.authConfig.ModuleAccountPermissions {
		maccPerms[permission.Account] = permission.Permissions
	}

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	cdc := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Amino

	maccPerms[authtypes.Burner] = []string{authtypes.Burner}
	maccPerms[authtypes.Minter] = []string{authtypes.Minter}

	s.Accountkeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(s.fetchStoreKey(banktypes.StoreKey).(*storetypes.KVStoreKey)),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(), authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	s.Bankkeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(s.fetchStoreKey(banktypes.StoreKey).(*storetypes.KVStoreKey)),
		s.Accountkeeper,
		blockedAddrs,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	s.Stakingkeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(s.fetchStoreKey(stakingtypes.StoreKey).(*storetypes.KVStoreKey)),
		s.Accountkeeper,
		s.Bankkeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	s.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		cdc,
		runtime.NewKVStoreService(s.fetchStoreKey(banktypes.StoreKey).(*storetypes.KVStoreKey)),
		s.Stakingkeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	s.Registrykeeper = registrykeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(s.fetchStoreKey(registrytypes.StoreKey).(*storetypes.KVStoreKey)),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	s.distrKeeper = distrkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(s.fetchStoreKey(distrtypes.StoreKey).(*storetypes.KVStoreKey)), s.Accountkeeper, s.Bankkeeper, s.Stakingkeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	s.Reporterkeeper = reporterkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(s.fetchStoreKey(reportertypes.StoreKey).(*storetypes.KVStoreKey)), log.NewNopLogger(), authtypes.NewModuleAddress(govtypes.ModuleName).String(), s.Stakingkeeper, s.Bankkeeper, s.Registrykeeper,
	)
	s.Oraclekeeper = oraclekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(s.fetchStoreKey(oracletypes.StoreKey).(*storetypes.KVStoreKey)), s.Accountkeeper, s.Bankkeeper, s.Registrykeeper, s.Reporterkeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	s.Disputekeeper = disputekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(s.fetchStoreKey(disputetypes.StoreKey).(*storetypes.KVStoreKey)), s.Accountkeeper, s.Bankkeeper, s.Oraclekeeper, s.Reporterkeeper,
	)
	s.Mintkeeper = mintkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(s.fetchStoreKey(minttypes.StoreKey).(*storetypes.KVStoreKey)), s.Accountkeeper, s.Bankkeeper,
	)
	s.Stakingkeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			s.distrKeeper.Hooks(),
			s.Reporterkeeper.Hooks(),
		),
	)
	s.Registrykeeper.SetHooks(
		registrytypes.NewMultiRegistryHooks(
			s.Oraclekeeper.Hooks(),
		),
	)
}

func (s *SharedSetup) SetupTest(t *testing.T) {
	t.Helper()
	s.require = require.New(t)
	sdk.DefaultBondDenom = "loya"
	config.SetupConfig()

	app, err := simtestutil.SetupWithConfiguration(
		depinject.Configs(
			configurator.NewAppConfig(
				AuthModule(),
				configurator.BankModule(),
				configurator.StakingModule(),
				configurator.SlashingModule(),
				configurator.ParamsModule(),
				configurator.ConsensusModule(),
				configurator.DistributionModule(),
				OracleModule(),
				DisputeModule(),
				RegistryModule(),
				MintModule(),
				ReporterModule(),
				configurator.GovModule(),
			),
			depinject.Supply(log.NewNopLogger()),
		),
		DefaultStartUpConfig(),
		&s.Accountkeeper, &s.Bankkeeper, &s.Stakingkeeper, &s.slashingKeeper,
		&s.interfaceRegistry, &s.appCodec, &s.authConfig, &s.Oraclekeeper,
		&s.Disputekeeper, &s.Registrykeeper, &s.Govkeeper, &s.distrKeeper, &s.Mintkeeper, &s.Reporterkeeper)

	s.require.NoError(err)
	s.Ctx = sdk.UnwrapSDKContext(app.BaseApp.NewContextLegacy(false, tmproto.Header{Time: time.Now()}))
	s.fetchStoreKey = app.UnsafeFindStoreKey

	s.queryHelper = baseapp.NewQueryServerTestHelper(s.Ctx, s.interfaceRegistry)
	s.Denom = sdk.DefaultBondDenom
	s.initKeepersWithmAccPerms(make(map[string]bool))
	s.App = app
}

func (s *SharedSetup) CreateValidators(numValidators int) ([]sdk.AccAddress, []sdk.ValAddress, []stakingtypes.Validator) {
	require := s.require

	// create account that will become a validator
	accountsAddrs := simtestutil.CreateIncrementalAccounts(numValidators)
	// mint numTrb for each validator
	initCoins := sdk.NewCoin(s.Denom, math.NewInt(5000*1e6))
	for _, acc := range accountsAddrs {
		// mint to module
		require.NoError(s.Bankkeeper.MintCoins(s.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		// send from module to account
		require.NoError(s.Bankkeeper.SendCoinsFromModuleToAccount(s.Ctx, authtypes.Minter, acc, sdk.NewCoins(initCoins)))
		require.Equal(initCoins, s.Bankkeeper.GetBalance(s.Ctx, acc, s.Denom))
	}

	// get val address for each account
	validatorsAddrs := simtestutil.ConvertAddrsToValAddrs(accountsAddrs)
	// create pub keys for validators
	pubKeys := simtestutil.CreateTestPubKeys(numValidators)
	validators := make([]stakingtypes.Validator, numValidators)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	// set each account with proper keepers
	for i, pubKey := range pubKeys {
		s.Accountkeeper.NewAccountWithAddress(s.Ctx, accountsAddrs[i])
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			validatorsAddrs[i].String(),
			pubKey,
			sdk.NewInt64Coin(s.Denom, 100),
			stakingtypes.Description{Moniker: strconv.Itoa(i)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			math.OneInt())
		require.NoError(err)

		_, err = stakingServer.CreateValidator(s.Ctx, valMsg)
		require.NoError(err)

		val, err := s.Stakingkeeper.GetValidator(s.Ctx, validatorsAddrs[i])
		require.NoError(err)
		s.MintTokens(accountsAddrs[i], math.NewInt(5000*1e6))
		msg := stakingtypes.MsgDelegate{DelegatorAddress: accountsAddrs[i].String(), ValidatorAddress: val.OperatorAddress, Amount: sdk.NewCoin(s.Denom, math.NewInt(5000*1e6))}
		_, err = stakingServer.Delegate(s.Ctx, &msg)
		require.NoError(err)
	}
	_, err := s.Stakingkeeper.EndBlocker(s.Ctx)
	require.NoError(err)
	return accountsAddrs, validatorsAddrs, validators
}

func (s *SharedSetup) MintTokens(addr sdk.AccAddress, amount math.Int) {
	require := s.require
	Ctx := s.Ctx
	coins := sdk.NewCoins(sdk.NewCoin(s.Denom, amount))
	require.NoError(s.Bankkeeper.MintCoins(Ctx, authtypes.Minter, coins))
	require.NoError(s.Bankkeeper.SendCoinsFromModuleToAccount(Ctx, authtypes.Minter, addr, coins))
}

func (s *SharedSetup) ConvertToAccAddress(priv []ed25519.PrivKey) []sdk.AccAddress {
	testAddrs := make([]sdk.AccAddress, len(priv))
	for i, pk := range priv {
		testAddrs[i] = sdk.AccAddress(pk.PubKey().Address())
	}
	return testAddrs
}

func (s *SharedSetup) CreateValidator(numValidators int) ([]sdk.AccAddress, []sdk.ValAddress, []stakingtypes.Validator) {
	require := s.require

	// create account that will become a validator
	accountsAddrs := simtestutil.CreateIncrementalAccounts(numValidators)
	// mint numTrb for each validator
	initCoins := sdk.NewCoin(s.Denom, math.NewInt(5000*1e6))
	for _, acc := range accountsAddrs {
		// mint to module
		require.NoError(s.Bankkeeper.MintCoins(s.Ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		// send from module to account
		require.NoError(s.Bankkeeper.SendCoinsFromModuleToAccount(s.Ctx, authtypes.Minter, acc, sdk.NewCoins(initCoins)))
		require.Equal(initCoins, s.Bankkeeper.GetBalance(s.Ctx, acc, s.Denom))
	}

	// get val address for each account
	validatorsAddrs := simtestutil.ConvertAddrsToValAddrs(accountsAddrs)
	// create pub keys for validators
	pubKeys := simtestutil.CreateTestPubKeys(numValidators)
	validators := make([]stakingtypes.Validator, numValidators)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	// set each account with proper keepers
	for i, pubKey := range pubKeys {
		s.Accountkeeper.NewAccountWithAddress(s.Ctx, accountsAddrs[i])
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			validatorsAddrs[i].String(),
			pubKey,
			sdk.NewInt64Coin(s.Denom, 100),
			stakingtypes.Description{Moniker: strconv.Itoa(i)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			math.OneInt())
		require.NoError(err)

		_, err = stakingServer.CreateValidator(s.Ctx, valMsg)
		require.NoError(err)
	}
	_, err := s.Stakingkeeper.EndBlocker(s.Ctx)
	require.NoError(err)
	return accountsAddrs, validatorsAddrs, validators
}
