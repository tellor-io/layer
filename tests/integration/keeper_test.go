package integration_test

import (
	"encoding/hex"
	"math/big"
	"time"

	"cosmossdk.io/math"

	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil/configurator"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

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
	"github.com/ethereum/go-ethereum/accounts/abi"
	bigmath "github.com/ethereum/go-ethereum/common/math"
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
		appCodec, suite.fetchStoreKey(banktypes.StoreKey), authtypes.ProtoBaseAccount,
		maccPerms, sdk.Bech32MainPrefix, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	suite.bankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, suite.fetchStoreKey(banktypes.StoreKey), suite.accountKeeper, blockedAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.stakingKeeper = stakingkeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(stakingtypes.StoreKey), suite.accountKeeper, suite.bankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.slashingKeeper = slashingkeeper.NewKeeper(
		appCodec, cdc, suite.fetchStoreKey(banktypes.StoreKey), suite.stakingKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	suite.registrykeeper = *registrykeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(registrytypes.StoreKey), suite.fetchStoreKey(registrytypes.StoreKey), paramtypes.Subspace{},
	)

	suite.oraclekeeper = *oraclekeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(oracletypes.StoreKey), suite.fetchStoreKey(oracletypes.StoreKey), paramtypes.Subspace{}, suite.accountKeeper, suite.bankKeeper, suite.stakingKeeper, suite.registrykeeper,
	)

	suite.disputekeeper = *disputekeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(disputetypes.StoreKey), suite.fetchStoreKey(disputetypes.StoreKey), paramtypes.Subspace{}, suite.accountKeeper, suite.bankKeeper, suite.oraclekeeper, suite.slashingKeeper, suite.stakingKeeper,
	)
}

func (s *IntegrationTestSuite) SetupTest() {
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

	app, err := sims.SetupWithConfiguration(
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
		integration.DefaultStartUpConfig(),
		&s.accountKeeper, &s.bankKeeper, &s.stakingKeeper, &s.slashingKeeper,
		&s.interfaceRegistry, &s.appCodec, &s.authConfig, &s.oraclekeeper,
		&s.disputekeeper, &s.registrykeeper)

	s.NoError(err)
	s.ctx = sdk.UnwrapSDKContext(app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()}))
	s.fetchStoreKey = app.UnsafeFindStoreKey

	s.queryHelper = baseapp.NewQueryServerTestHelper(s.ctx, s.interfaceRegistry)
	s.denom = sdk.DefaultBondDenom
	s.initKeepersWithmAccPerms(make(map[string]bool))
	s.app = app
}

func (s *IntegrationTestSuite) mintTokens(addr sdk.AccAddress, amount sdk.Coin) {
	ctx := s.ctx
	s.accountKeeper.SetAccount(ctx, authtypes.NewBaseAccountWithAddress(addr))
	s.NoError(s.bankKeeper.MintCoins(ctx, authtypes.Minter, sdk.NewCoins(amount)))
	s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, addr, sdk.NewCoins(amount)))
}

func (s *IntegrationTestSuite) newKeysWithTokens() sdk.AccAddress {
	PrivKey := secp256k1.GenPrivKey()
	PubKey := PrivKey.PubKey()
	Addr := sdk.AccAddress(PubKey.Address())
	s.mintTokens(Addr, sdk.NewCoin(s.denom, sdk.NewInt(1000000)))
	return Addr
}

func (s *IntegrationTestSuite) microReport() (disputetypes.MicroReport, sdk.ValAddress) {
	val := s.stakingKeeper.GetAllValidators(s.ctx)[0]
	valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
	s.Require().NoError(err)
	return disputetypes.MicroReport{
		Reporter:  sdk.AccAddress(valAddr).String(),
		Power:     val.GetConsensusPower(val.GetBondedTokens()),
		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
		Timestamp: 1696516597,
	}, valAddr

}

func (s *IntegrationTestSuite) createValidators(powers []int64) ([]sdk.AccAddress, []sdk.ValAddress) {
	ctx := s.ctx
	acctNum := len(powers)
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	amount := new(big.Int).Mul(big.NewInt(1000), base)
	testAddrs := simtestutil.CreateIncrementalAccounts(acctNum)
	addrs := s.addTestAddrs(acctNum, math.NewIntFromBigInt(amount), testAddrs)
	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrs)
	pks := simtestutil.CreateTestPubKeys(acctNum)

	for i, pk := range pks {
		val, err := stakingtypes.NewValidator(valAddrs[i], pk, stakingtypes.Description{})
		s.NoError(err)
		s.stakingKeeper.SetValidator(ctx, val)
		s.stakingKeeper.SetValidatorByConsAddr(ctx, val)
		s.stakingKeeper.SetNewValidatorByPowerIndex(ctx, val)
		s.stakingKeeper.Delegate(ctx, addrs[i], s.stakingKeeper.TokensFromConsensusPower(ctx, powers[i]), stakingtypes.Unbonded, val, true)
	}

	_ = staking.EndBlocker(ctx, s.stakingKeeper)

	return addrs, valAddrs
}

func (s *IntegrationTestSuite) addTestAddrs(accNum int, accAmt math.Int, testAddrs []sdk.AccAddress) []sdk.AccAddress {
	initCoins := sdk.NewCoin(s.denom, accAmt)
	for _, addr := range testAddrs {
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, addr, sdk.NewCoins(initCoins)))
	}

	return testAddrs
}

type ModuleAccs struct {
	staking authtypes.AccountI
	dispute authtypes.AccountI
}

func (s *IntegrationTestSuite) ModuleAccs() ModuleAccs {
	return ModuleAccs{
		staking: s.accountKeeper.GetModuleAccount(s.ctx, "bonded_tokens_pool"),
		dispute: s.accountKeeper.GetModuleAccount(s.ctx, "dispute"),
	}
}

func CreateRandomPrivateKeys(accNum int) []secp256k1.PrivKey {
	testAddrs := make([]secp256k1.PrivKey, accNum)
	for i := 0; i < accNum; i++ {
		pk := secp256k1.GenPrivKey()
		testAddrs[i] = *pk
	}
	return testAddrs
}

func (s *IntegrationTestSuite) convertToAccAddress(priv []secp256k1.PrivKey) []sdk.AccAddress {
	testAddrs := make([]sdk.AccAddress, len(priv))
	for i, pk := range priv {
		testAddrs[i] = sdk.AccAddress(pk.PubKey().Address())
	}
	return testAddrs
}

func (s *IntegrationTestSuite) createValidatorAccs(powers []int64) ([]sdk.AccAddress, []sdk.ValAddress, []secp256k1.PrivKey) {
	ctx := s.ctx
	acctNum := len(powers)
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	amount := new(big.Int).Mul(big.NewInt(1000), base)
	privKeys := CreateRandomPrivateKeys(acctNum)
	testAddrs := s.convertToAccAddress(privKeys)
	addrs := s.addTestAddrs(acctNum, math.NewIntFromBigInt(amount), testAddrs)
	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrs)

	for i, pk := range privKeys {
		account := authtypes.BaseAccount{
			Address: testAddrs[i].String(),
			PubKey:  codectypes.UnsafePackAny(pk.PubKey()),
		}
		s.accountKeeper.SetAccount(s.ctx, &account)
		val, err := stakingtypes.NewValidator(valAddrs[i], pk.PubKey(), stakingtypes.Description{})
		s.NoError(err)
		s.stakingKeeper.SetValidator(ctx, val)
		s.stakingKeeper.SetValidatorByConsAddr(ctx, val)
		s.stakingKeeper.SetNewValidatorByPowerIndex(ctx, val)
		s.stakingKeeper.Delegate(ctx, addrs[i], s.stakingKeeper.TokensFromConsensusPower(ctx, powers[i]), stakingtypes.Unbonded, val, true)
	}

	_ = staking.EndBlocker(ctx, s.stakingKeeper)

	return addrs, valAddrs, privKeys
}

// helper to encode spots to hex string similar to evm tellor. works only with two decimal places ie 123.45
func encodeValue(number *big.Int) string {
	bigNumber := bigmath.BigPow(10, 18).Mul(number, big.NewInt(10_000_000_000_000_000)) // cover the two decimal places
	// Define the ABI type for uint256
	uint256ABIType, _ := abi.NewType("uint256", "uint256", nil)

	// Create an ABI Argument for the uint256 type
	arg := abi.Argument{
		Type: uint256ABIType,
	}

	// Pack the scaled integer using the ABI Argument
	encodedBytes, _ := abi.Arguments{arg}.Pack(bigNumber)
	// Convert the encoded bytes to a hex string
	encodedString := hex.EncodeToString(encodedBytes)
	return encodedString
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
