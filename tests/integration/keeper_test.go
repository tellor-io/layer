package integration_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	integration "github.com/tellor-io/layer/tests"
	"github.com/tellor-io/layer/testutil/sample"
	_ "github.com/tellor-io/layer/x/dispute"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	_ "github.com/tellor-io/layer/x/oracle"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	_ "github.com/tellor-io/layer/x/registry/module"
	registrytypes "github.com/tellor-io/layer/x/registry/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	_ "github.com/tellor-io/layer/x/reporter/module"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
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

var (
	ethQueryData, _ = hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	btcQueryData, _ = hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
	trbQueryData, _ = hex.DecodeString("00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000")
)

type IntegrationTestSuite struct {
	suite.Suite

	oraclekeeper   oraclekeeper.Keeper
	disputekeeper  disputekeeper.Keeper
	registrykeeper registrykeeper.Keeper
	reporterkeeper reporterkeeper.Keeper

	accountKeeper  authkeeper.AccountKeeper
	bankKeeper     bankkeeper.BaseKeeper
	distrKeeper    distrkeeper.Keeper
	slashingKeeper slashingkeeper.Keeper
	stakingKeeper  *stakingkeeper.Keeper
	govKeeper      *govkeeper.Keeper
	ctx            sdk.Context
	appCodec       codec.Codec
	authConfig     *authmodulev1.Module

	interfaceRegistry codectypes.InterfaceRegistry
	fetchStoreKey     func(string) storetypes.StoreKey

	denom string
	app   *runtime.App
}

func (suite *IntegrationTestSuite) initKeepersWithmAccPerms(blockedAddrs map[string]bool) {
	maccPerms := map[string][]string{}
	for _, permission := range suite.authConfig.ModuleAccountPermissions {
		maccPerms[permission.Account] = permission.Permissions
	}

	appCodec := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Codec
	cdc := moduletestutil.MakeTestEncodingConfig(auth.AppModuleBasic{}, bank.AppModuleBasic{}, staking.AppModuleBasic{}).Amino

	maccPerms[authtypes.Burner] = []string{authtypes.Burner}
	maccPerms[authtypes.Minter] = []string{authtypes.Minter}

	suite.accountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(suite.fetchStoreKey(banktypes.StoreKey).(*storetypes.KVStoreKey)),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(), authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.bankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(suite.fetchStoreKey(banktypes.StoreKey).(*storetypes.KVStoreKey)),
		suite.accountKeeper,
		blockedAddrs,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	suite.stakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(suite.fetchStoreKey(stakingtypes.StoreKey).(*storetypes.KVStoreKey)),
		suite.accountKeeper,
		suite.bankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	suite.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		cdc,
		runtime.NewKVStoreService(suite.fetchStoreKey(banktypes.StoreKey).(*storetypes.KVStoreKey)),
		suite.stakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.registrykeeper = registrykeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(suite.fetchStoreKey(registrytypes.StoreKey).(*storetypes.KVStoreKey)),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.distrKeeper = distrkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(suite.fetchStoreKey(distrtypes.StoreKey).(*storetypes.KVStoreKey)), suite.accountKeeper, suite.bankKeeper, suite.stakingKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.reporterkeeper = reporterkeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(suite.fetchStoreKey(reportertypes.StoreKey).(*storetypes.KVStoreKey)), log.NewNopLogger(), authtypes.NewModuleAddress(govtypes.ModuleName).String(), suite.stakingKeeper, suite.bankKeeper,
	)

	suite.oraclekeeper = oraclekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(suite.fetchStoreKey(oracletypes.StoreKey).(*storetypes.KVStoreKey)), suite.accountKeeper, suite.bankKeeper, suite.registrykeeper, suite.reporterkeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	suite.disputekeeper = disputekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(suite.fetchStoreKey(disputetypes.StoreKey).(*storetypes.KVStoreKey)), suite.accountKeeper, suite.bankKeeper, suite.oraclekeeper, suite.reporterkeeper,
	)
	suite.stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			suite.reporterkeeper.Hooks(),
		),
	)
	suite.registrykeeper.SetHooks(
		registrytypes.NewMultiRegistryHooks(
			suite.oraclekeeper.Hooks(),
		),
	)
}

func (s *IntegrationTestSuite) SetupTest() {
	sdk.DefaultBondDenom = "loya"
	config.SetupConfig()

	app, err := simtestutil.SetupWithConfiguration(
		depinject.Configs(
			configurator.NewAppConfig(
				integration.AuthModule(),
				configurator.BankModule(),
				configurator.StakingModule(),
				configurator.SlashingModule(),
				configurator.ParamsModule(),
				configurator.ConsensusModule(),
				configurator.DistributionModule(),
				integration.OracleModule(),
				integration.DisputeModule(),
				integration.RegistryModule(),
				integration.ReporterModule(),
				configurator.GovModule(),
			),
			depinject.Supply(log.NewNopLogger()),
		),
		integration.DefaultStartUpConfig(),
		&s.accountKeeper, &s.bankKeeper, &s.stakingKeeper, &s.slashingKeeper,
		&s.interfaceRegistry, &s.appCodec, &s.authConfig, &s.oraclekeeper,
		&s.disputekeeper, &s.registrykeeper, &s.govKeeper, &s.reporterkeeper)

	s.NoError(err)
	s.ctx = sdk.UnwrapSDKContext(app.BaseApp.NewContextLegacy(false, tmproto.Header{Time: time.Now()}))
	s.fetchStoreKey = app.UnsafeFindStoreKey

	s.denom = sdk.DefaultBondDenom
	s.initKeepersWithmAccPerms(make(map[string]bool))
	s.app = app
}

func (s *IntegrationTestSuite) mintTokens(addr sdk.AccAddress, amount math.Int) {
	ctx := s.ctx
	// s.accountKeeper.SetAccount(ctx, authtypes.NewBaseAccountWithAddress(addr))
	coins := sdk.NewCoins(sdk.NewCoin(s.denom, amount))
	s.NoError(s.bankKeeper.MintCoins(ctx, authtypes.Minter, coins))
	s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, addr, coins))
}

func (s *IntegrationTestSuite) newKeysWithTokens() sdk.AccAddress {
	Addr := sample.AccAddressBytes()
	s.mintTokens(Addr, math.NewInt(1_000_000))
	return Addr
}

// func (s *IntegrationTestSuite) createValidators(powers []int64) ([]sdk.AccAddress, []sdk.ValAddress) {
// 	ctx := s.ctx
// 	acctNum := len(powers)
// 	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
// 	amount := new(big.Int).Mul(big.NewInt(1000), base)
// 	testAddrs := simtestutil.CreateIncrementalAccounts(acctNum)
// 	addrs := s.addTestAddrs(math.NewIntFromBigInt(amount), testAddrs)
// 	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrs)
// 	pks := simtestutil.CreateTestPubKeys(acctNum)

// 	for i, pk := range pks {
// 		s.accountKeeper.NewAccountWithAddress(ctx, testAddrs[i])

// 		val, err := stakingtypes.NewValidator(valAddrs[i].String(), pk, stakingtypes.Description{})
// 		s.NoError(err)
// 		s.stakingKeeper.SetValidator(ctx, val)
// 		s.stakingKeeper.SetValidatorByConsAddr(ctx, val)
// 		s.stakingKeeper.SetNewValidatorByPowerIndex(ctx, val)
// 		s.stakingKeeper.Delegate(ctx, addrs[i], s.stakingKeeper.TokensFromConsensusPower(ctx, powers[i]), stakingtypes.Unbonded, val, true)
// 	}

// 	_, err := s.stakingKeeper.EndBlocker(ctx)
// 	s.NoError(err)

// 	return addrs, valAddrs
// }

func (s *IntegrationTestSuite) addTestAddrs(accAmt math.Int, testAddrs []sdk.AccAddress) []sdk.AccAddress {
	initCoins := sdk.NewCoin(s.denom, accAmt)
	for _, addr := range testAddrs {
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, addr, sdk.NewCoins(initCoins)))
	}

	return testAddrs
}

type ModuleAccs struct {
	staking sdk.AccountI
	dispute sdk.AccountI
}

func (s *IntegrationTestSuite) ModuleAccs() ModuleAccs {
	return ModuleAccs{
		staking: s.accountKeeper.GetModuleAccount(s.ctx, "bonded_tokens_pool"),
		dispute: s.accountKeeper.GetModuleAccount(s.ctx, "dispute"),
	}
}

func CreateRandomPrivateKeys(accNum int) []ed25519.PrivKey {
	testAddrs := make([]ed25519.PrivKey, accNum)
	for i := 0; i < accNum; i++ {
		pk := ed25519.GenPrivKey()
		testAddrs[i] = *pk
	}
	return testAddrs
}

func (s *IntegrationTestSuite) convertToAccAddress(priv []ed25519.PrivKey) []sdk.AccAddress {
	testAddrs := make([]sdk.AccAddress, len(priv))
	for i, pk := range priv {
		testAddrs[i] = sdk.AccAddress(pk.PubKey().Address())
	}
	return testAddrs
}

func (s *IntegrationTestSuite) createValidatorAccs(powers []int64) ([]sdk.AccAddress, []sdk.ValAddress, []ed25519.PrivKey) {
	ctx := s.ctx
	acctNum := len(powers)
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	amount := new(big.Int).Mul(big.NewInt(1000), base)
	privKeys := CreateRandomPrivateKeys(acctNum)
	testAddrs := s.convertToAccAddress(privKeys)
	addrs := s.addTestAddrs(math.NewIntFromBigInt(amount), testAddrs)
	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrs)

	for i, pk := range privKeys {
		account := authtypes.BaseAccount{
			Address:       testAddrs[i].String(),
			PubKey:        codectypes.UnsafePackAny(pk.PubKey()),
			AccountNumber: uint64(i + 1),
		}
		s.accountKeeper.SetAccount(s.ctx, &account)
		val, err := stakingtypes.NewValidator(valAddrs[i].String(), pk.PubKey(), stakingtypes.Description{})
		s.NoError(err)
		s.NoError(s.stakingKeeper.SetValidator(ctx, val))
		s.NoError(s.stakingKeeper.SetValidatorByConsAddr(ctx, val))
		s.NoError(s.stakingKeeper.SetNewValidatorByPowerIndex(ctx, val))
		_, err = s.stakingKeeper.Delegate(ctx, addrs[i], s.stakingKeeper.TokensFromConsensusPower(ctx, powers[i]), stakingtypes.Unbonded, val, true)
		s.NoError(err)
		// call hooks for distribution init
		valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		err = s.distrKeeper.Hooks().AfterValidatorCreated(ctx, valBz)
		s.NoError(err)
		err = s.distrKeeper.Hooks().BeforeDelegationCreated(ctx, addrs[i], valBz)
		s.NoError(err)
		err = s.distrKeeper.Hooks().AfterDelegationModified(ctx, addrs[i], valBz)
		s.NoError(err)
	}

	_, err := s.stakingKeeper.EndBlocker(ctx)
	s.NoError(err)

	return addrs, valAddrs, privKeys
}

func (s *IntegrationTestSuite) CreateAccountsWithTokens(numofAccs int, amountOfTokens int64) []sdk.AccAddress {
	privKeys := CreateRandomPrivateKeys(numofAccs)
	accs := make([]sdk.AccAddress, numofAccs)
	for i, pk := range privKeys {
		accs[i] = sdk.AccAddress(pk.PubKey().Address())
		s.mintTokens(accs[i], math.NewInt(amountOfTokens))
	}
	return accs
}

// inspired by telliot python code
func encodeValue(number float64) string {
	strNumber := fmt.Sprintf("%.18f", number)

	parts := strings.Split(strNumber, ".")
	if len(parts[1]) > 18 {
		parts[1] = parts[1][:18]
	}
	truncatedStr := parts[0] + parts[1]

	bigIntNumber := new(big.Int)
	bigIntNumber.SetString(truncatedStr, 10)

	uint256ABIType, _ := abi.NewType("uint256", "", nil)

	arguments := abi.Arguments{{Type: uint256ABIType}}
	encodedBytes, _ := arguments.Pack(bigIntNumber)

	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
