package integration_test

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
	setup "github.com/tellor-io/layer/tests"
	"github.com/tellor-io/layer/testutil/sample"
	_ "github.com/tellor-io/layer/x/dispute"
	_ "github.com/tellor-io/layer/x/oracle"
	_ "github.com/tellor-io/layer/x/registry/module"
	_ "github.com/tellor-io/layer/x/reporter/module"

	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	_ "github.com/cosmos/cosmos-sdk/x/genutil"
	_ "github.com/cosmos/cosmos-sdk/x/gov"
	_ "github.com/cosmos/cosmos-sdk/x/mint"
	_ "github.com/cosmos/cosmos-sdk/x/params"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
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
	Setup *setup.SharedSetup
}

func (s *IntegrationTestSuite) SetupTest() {
	s.Setup = &setup.SharedSetup{}
	s.Setup.SetupTest(s.T())
}

func (s *IntegrationTestSuite) newKeysWithTokens() sdk.AccAddress {
	Addr := sample.AccAddressBytes()
	s.Setup.MintTokens(Addr, math.NewInt(1_000_000))
	return Addr
}

func CreateRandomPrivateKeys(accNum int) []ed25519.PrivKey {
	testAddrs := make([]ed25519.PrivKey, accNum)
	for i := 0; i < accNum; i++ {
		pk := ed25519.GenPrivKey()
		testAddrs[i] = *pk
	}
	return testAddrs
}

// todo: remove this
func (s *IntegrationTestSuite) createValidatorAccs(powers []uint64) ([]sdk.AccAddress, []sdk.ValAddress, []ed25519.PrivKey) {
	ctx := s.Setup.Ctx
	acctNum := len(powers)
	base := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	amount := new(big.Int).Mul(big.NewInt(1000), base)
	privKeys := CreateRandomPrivateKeys(acctNum)
	testAddrs := s.Setup.ConvertToAccAddress(privKeys)
	for _, addr := range testAddrs {
		s.Setup.MintTokens(addr, math.NewIntFromBigInt(amount))
	}
	valAddrs := simtestutil.ConvertAddrsToValAddrs(testAddrs)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	for i, pk := range privKeys {
		account := authtypes.BaseAccount{
			Address:       testAddrs[i].String(),
			PubKey:        codectypes.UnsafePackAny(pk.PubKey()),
			AccountNumber: uint64(i + 100),
		}
		s.Setup.Accountkeeper.SetAccount(s.Setup.Ctx, &account)
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			valAddrs[i].String(),
			pk.PubKey(),
			sdk.NewInt64Coin(s.Setup.Denom, s.Setup.Stakingkeeper.TokensFromConsensusPower(ctx, int64(powers[i])).Int64()),
			stakingtypes.Description{Moniker: strconv.Itoa(i)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			math.OneInt())
		s.NoError(err)

		_, err = stakingServer.CreateValidator(s.Setup.Ctx, valMsg)
		s.NoError(err)

		val, err := s.Setup.Stakingkeeper.GetValidator(ctx, valAddrs[i])
		s.NoError(err)
		s.Setup.MintTokens(testAddrs[i], s.Setup.Stakingkeeper.TokensFromConsensusPower(ctx, int64(powers[i])))
		msg := stakingtypes.MsgDelegate{DelegatorAddress: testAddrs[i].String(), ValidatorAddress: val.OperatorAddress, Amount: sdk.NewCoin(s.Setup.Denom, s.Setup.Stakingkeeper.TokensFromConsensusPower(ctx, int64(powers[i])))}
		_, err = stakingServer.Delegate(s.Setup.Ctx, &msg)
		s.NoError(err)
	}

	_, err := s.Setup.Stakingkeeper.EndBlocker(ctx)
	s.NoError(err)

	return testAddrs, valAddrs, privKeys
}

func (s *IntegrationTestSuite) CreateAccountsWithTokens(numofAccs int, amountOfTokens int64) []sdk.AccAddress {
	privKeys := CreateRandomPrivateKeys(numofAccs)
	accs := make([]sdk.AccAddress, numofAccs)
	for i, pk := range privKeys {
		accs[i] = sdk.AccAddress(pk.PubKey().Address())
		s.Setup.MintTokens(accs[i], math.NewInt(amountOfTokens))
	}
	return accs
}

func (s *IntegrationTestSuite) createValidatorsbypowers(powers []uint64) ([]sdk.AccAddress, []sdk.ValAddress, []ed25519.PrivKey) {
	ctx := s.Setup.Ctx
	acctNum := len(powers)
	privKeys := CreateRandomPrivateKeys(acctNum)
	testAddrs := s.Setup.ConvertToAccAddress(privKeys)
	for i, addr := range testAddrs {
		s.Setup.MintTokens(addr, math.NewInt(int64(powers[i]*1000000)))
	}
	valAddrs := simtestutil.ConvertAddrsToValAddrs(testAddrs)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	for i, pk := range privKeys {
		account := authtypes.BaseAccount{
			Address:       testAddrs[i].String(),
			PubKey:        codectypes.UnsafePackAny(pk.PubKey()),
			AccountNumber: uint64(i + 100),
		}
		s.Setup.Accountkeeper.SetAccount(s.Setup.Ctx, &account)
		valMsg, err := stakingtypes.NewMsgCreateValidator(
			valAddrs[i].String(),
			pk.PubKey(),
			sdk.NewInt64Coin(s.Setup.Denom, s.Setup.Stakingkeeper.TokensFromConsensusPower(ctx, int64(powers[i])).Int64()),
			stakingtypes.Description{Moniker: strconv.Itoa(i)},
			stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			math.OneInt())
		s.NoError(err)

		_, err = stakingServer.CreateValidator(s.Setup.Ctx, valMsg)
		s.NoError(err)
	}

	_, err := s.Setup.Stakingkeeper.EndBlocker(ctx)
	s.NoError(err)

	return testAddrs, valAddrs, privKeys
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

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
