package e2e_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/suite"
	"github.com/tellor-io/layer/app/config"
	testutils "github.com/tellor-io/layer/tests"
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

	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/configurator"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	// _ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	// _ "github.com/cosmos/cosmos-sdk/x/bank"
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

const (
	ethQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	btcQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
	trbQueryData = "0x00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706F745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
)

type E2ETestSuite struct {
	suite.Suite

	oraclekeeper   oraclekeeper.Keeper
	disputekeeper  disputekeeper.Keeper
	registrykeeper registrykeeper.Keeper
	mintkeeper     mintkeeper.Keeper
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

	queryHelper       *baseapp.QueryServiceTestHelper
	interfaceRegistry codectypes.InterfaceRegistry
	fetchStoreKey     func(string) storetypes.StoreKey

	denom string
	app   *runtime.App
}

func (suite *E2ETestSuite) initKeepersWithmAccPerms(blockedAddrs map[string]bool) {
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
	suite.oraclekeeper = oraclekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(suite.fetchStoreKey(oracletypes.StoreKey).(*storetypes.KVStoreKey)), suite.accountKeeper, suite.bankKeeper, suite.registrykeeper, suite.reporterkeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	suite.disputekeeper = disputekeeper.NewKeeper(
		appCodec, runtime.NewKVStoreService(suite.fetchStoreKey(disputetypes.StoreKey).(*storetypes.KVStoreKey)), suite.accountKeeper, suite.bankKeeper, suite.oraclekeeper, suite.reporterkeeper,
	)
	suite.mintkeeper = mintkeeper.NewKeeper(
		appCodec, suite.fetchStoreKey(minttypes.StoreKey), suite.accountKeeper, suite.bankKeeper,
	)
}

func (s *E2ETestSuite) SetupTest() {
	sdk.DefaultBondDenom = "loya"
	config.SetupConfig()

	app, err := simtestutil.SetupWithConfiguration(
		depinject.Configs(
			configurator.NewAppConfig(
				testutils.AuthModule(),
				configurator.BankModule(),
				configurator.StakingModule(),
				configurator.SlashingModule(),
				configurator.ParamsModule(),
				configurator.ConsensusModule(),
				configurator.DistributionModule(),
				testutils.OracleModule(),
				testutils.DisputeModule(),
				testutils.RegistryModule(),
				testutils.MintModule(),
				testutils.ReporterModule(),
				configurator.GovModule(),
			),
			depinject.Supply(log.NewNopLogger()),
		),
		testutils.DefaultStartUpConfig(),
		&s.accountKeeper, &s.bankKeeper, &s.stakingKeeper, &s.slashingKeeper,
		&s.interfaceRegistry, &s.appCodec, &s.authConfig, &s.oraclekeeper,
		&s.disputekeeper, &s.registrykeeper, &s.govKeeper, &s.distrKeeper, &s.mintkeeper, &s.reporterkeeper)

	s.NoError(err)
	s.ctx = sdk.UnwrapSDKContext(app.BaseApp.NewContextLegacy(false, tmproto.Header{Time: time.Now()}))
	s.fetchStoreKey = app.UnsafeFindStoreKey

	s.queryHelper = baseapp.NewQueryServerTestHelper(s.ctx, s.interfaceRegistry)
	s.denom = sdk.DefaultBondDenom
	s.initKeepersWithmAccPerms(make(map[string]bool))
	s.app = app
}

func (s *E2ETestSuite) CreateValidators(numValidators int) ([]sdk.AccAddress, []sdk.ValAddress, []stakingtypes.Validator) {
	require := s.Require()

	// create account that will become a validator
	accountsAddrs := simtestutil.CreateIncrementalAccounts(numValidators)
	// mint numTrb for each validator
	initCoins := sdk.NewCoin(s.denom, math.NewInt(5000*1e6))
	for _, acc := range accountsAddrs {
		// mint to module
		require.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		// send from module to account
		require.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, acc, sdk.NewCoins(initCoins)))
		require.Equal(initCoins, s.bankKeeper.GetBalance(s.ctx, acc, s.denom))
	}

	// get val address for each account
	validatorsAddrs := simtestutil.ConvertAddrsToValAddrs(accountsAddrs)
	// create pub keys for validators
	pubKeys := simtestutil.CreateTestPubKeys(numValidators)
	validators := make([]stakingtypes.Validator, numValidators)
	// set each account with proper keepers
	for i, pubKey := range pubKeys {
		s.accountKeeper.NewAccountWithAddress(s.ctx, accountsAddrs[i])
		validator, err := stakingtypes.NewValidator(validatorsAddrs[i].String(), pubKey, stakingtypes.Description{Moniker: strconv.Itoa(i)})
		require.NoError(err)
		validators[i] = validator
		s.NoError(s.stakingKeeper.SetValidator(s.ctx, validator))
		s.NoError(s.stakingKeeper.SetValidatorByConsAddr(s.ctx, validator))
		s.NoError(s.stakingKeeper.SetNewValidatorByPowerIndex(s.ctx, validator))

		_, err = s.stakingKeeper.Delegate(s.ctx, accountsAddrs[i], math.NewInt(5000*1e6), stakingtypes.Unbonded, validator, true)
		require.NoError(err)
		// call hooks for distribution init
		valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
		if err != nil {
			panic(err)
		}
		err = s.distrKeeper.Hooks().AfterValidatorCreated(s.ctx, valBz)
		require.NoError(err)
		err = s.distrKeeper.Hooks().BeforeDelegationCreated(s.ctx, accountsAddrs[i], valBz)
		require.NoError(err)
		err = s.distrKeeper.Hooks().AfterDelegationModified(s.ctx, accountsAddrs[i], valBz)
		require.NoError(err)
	}

	return accountsAddrs, validatorsAddrs, validators
}

func (s *E2ETestSuite) CreateReporters(numReporters int, valAddrs []sdk.ValAddress, vals []stakingtypes.Validator) []sdk.AccAddress {
	require := s.Require()
	type Delegator struct {
		delegatorAddress sdk.AccAddress
		validator        stakingtypes.Validator
		tokenAmount      math.Int
	}

	if numReporters != len(valAddrs) {
		panic("numReporters must be equal to the the number of validators (make other reporters manually)")
	}

	msgServerReporter := reporterkeeper.NewMsgServerImpl(s.reporterkeeper)
	require.NotNil(msgServerReporter)

	// create reporter accounts from random private keys
	privKeys := CreateRandomPrivateKeys(numReporters)
	accs := s.convertToAccAddress(privKeys) // sdk.AccountAddresses

	// mint 1k trb to each account
	initCoins := sdk.NewCoin(s.denom, math.NewInt(1000*1e6))
	for _, acc := range accs {
		s.NoError(s.bankKeeper.MintCoins(s.ctx, authtypes.Minter, sdk.NewCoins(initCoins)))
		s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(s.ctx, authtypes.Minter, acc, sdk.NewCoins(initCoins)))
	}
	// delegate 1k trb to validators
	for i, acc := range accs {
		valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(vals[i].GetOperator())
		require.NoError(err)
		reporterDelToVal := Delegator{delegatorAddress: acc, validator: vals[i], tokenAmount: math.NewInt(1000 * 1e6)}
		_, err = s.stakingKeeper.Delegate(s.ctx, reporterDelToVal.delegatorAddress, reporterDelToVal.tokenAmount, stakingtypes.Unbonded, reporterDelToVal.validator, true)
		require.NoError(err)
		// call dist module hooks
		err = s.distrKeeper.Hooks().AfterValidatorCreated(s.ctx, valBz)
		require.NoError(err)
		err = s.distrKeeper.Hooks().BeforeDelegationCreated(s.ctx, acc, valBz)
		require.NoError(err)
		err = s.distrKeeper.Hooks().AfterDelegationModified(s.ctx, acc, valBz)
		require.NoError(err)
	}
	// self delegate in reporter module with 1k trb
	// CreateReporter tx
	for i, acc := range accs {
		valBz, err := s.stakingKeeper.ValidatorAddressCodec().StringToBytes(vals[i].GetOperator())
		require.NoError(err)
		commission := stakingtypes.NewCommissionWithTime(math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(3, 1),
			math.LegacyNewDecWithPrec(1, 1), s.ctx.BlockTime()) // rate = 10%, maxRate = 30%, maxChangeRate = 10%

		var createReporterMsg reportertypes.MsgCreateReporter
		createReporterMsg.Reporter = acc.String()
		createReporterMsg.Amount = math.NewInt(1000 * 1e6)
		createReporterMsg.Commission = &commission
		valAddr, err := sdk.ValAddressFromBech32(vals[i].GetOperator())
		require.NoError(err)
		createReporterMsg.TokenOrigins = []*reportertypes.TokenOrigin{
			{
				ValidatorAddress: valAddr.Bytes(),
				Amount:           math.NewInt(1000 * 1e6),
			},
		}
		// send CreateReporter Tx
		_, err = msgServerReporter.CreateReporter(s.ctx, &createReporterMsg)
		s.NoError(err)

		// verify in collections
		rkDelegation, err := s.reporterkeeper.Delegators.Get(s.ctx, acc)
		require.NoError(err)
		require.Equal(rkDelegation.Reporter, acc.Bytes())
		require.Equal(rkDelegation.Amount, math.NewInt(1000*1e6))
		// check on reporter/validator delegation
		skDelegation, err := s.stakingKeeper.Delegation(s.ctx, acc, valBz)
		require.NoError(err)
		require.Equal(skDelegation.GetDelegatorAddr(), acc.String())
		require.Equal(skDelegation.GetValidatorAddr(), vals[i].GetOperator())
	}

	return accs
}

func (s *E2ETestSuite) mintTokens(addr sdk.AccAddress, amount sdk.Coin) {
	ctx := s.ctx
	s.accountKeeper.SetAccount(ctx, authtypes.NewBaseAccountWithAddress(addr))
	s.NoError(s.bankKeeper.MintCoins(ctx, authtypes.Minter, sdk.NewCoins(amount)))
	s.NoError(s.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, addr, sdk.NewCoins(amount)))
}

// func (s *E2ETestSuite) microReport() (disputetypes.MicroReport, sdk.ValAddress) {
// 	vals, err := s.stakingKeeper.GetAllValidators(s.ctx)
// 	s.Require().NoError(err)
// 	valAddr, err := sdk.ValAddressFromBech32(vals[0].OperatorAddress)
// 	s.Require().NoError(err)
// 	return disputetypes.MicroReport{
// 		Reporter:  sdk.AccAddress(valAddr).String(),
// 		Power:     vals[0].GetConsensusPower(vals[0].GetBondedTokens()),
// 		QueryId:   "83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992",
// 		Value:     "000000000000000000000000000000000000000000000058528649cf80ee0000",
// 		Timestamp: 1696516597,
// 	}, valAddr

// }

func (s *E2ETestSuite) CreateAccountsWithTokens(numofAccs int, amountOfTokens int64) []sdk.AccAddress {
	privKeys := CreateRandomPrivateKeys(numofAccs)
	accs := make([]sdk.AccAddress, numofAccs)
	for i, pk := range privKeys {
		accs[i] = sdk.AccAddress(pk.PubKey().Address())
		s.mintTokens(accs[i], sdk.NewCoin(s.denom, math.NewInt(amountOfTokens)))
	}
	return accs
}

// func (s *E2ETestSuite) createValidators(powers []int64) ([]sdk.AccAddress, []sdk.ValAddress) {
// 	ctx := s.ctx
// 	acctNum := len(powers)
// 	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
// 	amount := new(big.Int).Mul(big.NewInt(1000), base)
// 	testAddrs := simtestutil.CreateIncrementalAccounts(acctNum)
// 	addrs := s.addTestAddrs(acctNum, math.NewIntFromBigInt(amount), testAddrs)
// 	valAddrs := simtestutil.ConvertAddrsToValAddrs(addrs)
// 	pks := simtestutil.CreateTestPubKeys(acctNum)

// 	for i, pk := range pks {
// 		// account := authtypes.BaseAccount{
// 		// 	Address:       testAddrs[i].String(),
// 		// 	PubKey:        codectypes.UnsafePackAny(pk),
// 		// 	AccountNumber: s.accountKeeper.NextAccountNumber(ctx),
// 		// }
// 		// s.accountKeeper.NewAccount(s.ctx, &account)

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

type ModuleAccs struct {
	staking sdk.AccountI
	dispute sdk.AccountI
}

func (s *E2ETestSuite) ModuleAccs() ModuleAccs {
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

func (s *E2ETestSuite) convertToAccAddress(priv []secp256k1.PrivKey) []sdk.AccAddress {
	testAddrs := make([]sdk.AccAddress, len(priv))
	for i, pk := range priv {
		testAddrs[i] = sdk.AccAddress(pk.PubKey().Address())
	}
	return testAddrs
}

func JailValidator(ctx sdk.Context, consensusAddress sdk.ConsAddress, validatorAddress sdk.ValAddress, k stakingkeeper.Keeper) error {
	validator, err := k.GetValidator(ctx, validatorAddress)
	if err != nil {
		return fmt.Errorf("validator %s not found", validatorAddress)
	}

	if validator.Jailed {
		return fmt.Errorf("validator %s is already jailed", validatorAddress)
	}

	err = k.Jail(ctx, consensusAddress)
	if err != nil {
		return err
	}

	return nil
}

// func CommitReport(ctx context.Context, accountAddr string, []queryData bytes, msgServerOracle oracletypes.MsgServer) error {
// 	// commit
// 	value := "000000000000000000000000000000000000000000000058528649cf80ee0000"
// 	valueDecoded, err := hex.DecodeString(value) // convert hex value to bytes
// 	if err != nil {
// 		return err
// 	}
// 	salt, err := utils.Salt(32)
// 	if err != nil {
// 		return err
// 	}
// 	hash := utils.CalculateCommitment(string(valueDecoded), salt)
// 	if err != nil {
// 		return err
// 	}
// 	// commit report with query data in cycle list
// 	commitreq := &oracletypes.MsgCommitReport{
// 		Creator:   accountAddr,
// 		QueryData: queryData,
// 		Hash:      hash,
// 	}
// 	_, err = msgServerOracle.CommitReport(ctx, commitreq)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

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
	suite.Run(t, new(E2ETestSuite))
}
