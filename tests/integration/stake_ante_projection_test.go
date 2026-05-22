package integration_test

import (
	"fmt"
	"testing"

	"github.com/tellor-io/layer/testutil/encoding"
	reporterante "github.com/tellor-io/layer/x/reporter/ante"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// runStakeAnte builds a tx and executes only the reporter ante decorator,
// leaving keeper state unchanged for parity comparisons.
func (s *IntegrationTestSuite) runStakeAnte(t *testing.T, msgs ...sdk.Msg) error {
	t.Helper()

	txBuilder := client.Context{}.
		WithTxConfig(encoding.GetTestEncodingCfg().TxConfig).
		TxConfig.NewTxBuilder()
	s.Require().NoError(txBuilder.SetMsgs(msgs...))

	decorator := reporterante.NewTrackStakeChangesDecorator(s.Setup.Reporterkeeper, s.Setup.Stakingkeeper)
	_, err := decorator.AnteHandle(s.Setup.Ctx, txBuilder.GetTx(), false, func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	})
	return err
}

// runStakingMsgsOnCache executes the same staking messages against real staking
// keeper state without committing, so tests can compare ante projection with the
// state staking would actually produce.
func (s *IntegrationTestSuite) runStakingMsgsOnCache(msgs ...sdk.Msg) (sdk.Context, error) {
	cacheCtx, _ := s.Setup.Ctx.CacheContext()
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *stakingtypes.MsgCreateValidator:
			if _, err := stakingServer.CreateValidator(cacheCtx, msg); err != nil {
				return cacheCtx, err
			}
		case *stakingtypes.MsgDelegate:
			if _, err := stakingServer.Delegate(cacheCtx, msg); err != nil {
				return cacheCtx, err
			}
		case *stakingtypes.MsgUndelegate:
			if _, err := stakingServer.Undelegate(cacheCtx, msg); err != nil {
				return cacheCtx, err
			}
		default:
			return cacheCtx, fmt.Errorf("unsupported staking msg type %T", msg)
		}
	}
	if _, err := s.Setup.Stakingkeeper.EndBlocker(cacheCtx); err != nil {
		return cacheCtx, err
	}
	return cacheCtx, nil
}

// setStakeTrackerToCurrentBonded aligns reporter's 5% tracker with the current
// staking total before each scenario mutates stake.
func (s *IntegrationTestSuite) setStakeTrackerToCurrentBonded() math.Int {
	totalBonded, err := s.Setup.Stakingkeeper.TotalBondedTokens(s.Setup.Ctx)
	s.Require().NoError(err)
	s.Require().NoError(s.Setup.Reporterkeeper.Tracker.Set(s.Setup.Ctx, reportertypes.StakeTracker{Amount: totalBonded}))
	return totalBonded
}

// newValidatorAccount creates and funds an account that can submit
// MsgCreateValidator in parity tests.
func (s *IntegrationTestSuite) newValidatorAccount(accountNumber uint64, tokens math.Int) (sdk.AccAddress, sdk.ValAddress, *ed25519.PrivKey) {
	privKey := ed25519.GenPrivKey()
	accAddr := sdk.AccAddress(privKey.PubKey().Address())
	valAddr := sdk.ValAddress(accAddr)
	s.Setup.Accountkeeper.SetAccount(s.Setup.Ctx, &authtypes.BaseAccount{
		Address:       accAddr.String(),
		PubKey:        nil,
		AccountNumber: accountNumber,
	})
	s.Setup.MintTokens(accAddr, tokens)
	return accAddr, valAddr, privKey
}

// newAccount creates a funded delegator account without relying on global test
// account ordering.
func (s *IntegrationTestSuite) newAccount(accountNumber uint64, tokens math.Int) sdk.AccAddress {
	privKey := ed25519.GenPrivKey()
	accAddr := sdk.AccAddress(privKey.PubKey().Address())
	s.Setup.Accountkeeper.SetAccount(s.Setup.Ctx, &authtypes.BaseAccount{
		Address:       accAddr.String(),
		AccountNumber: accountNumber,
	})
	s.Setup.MintTokens(accAddr, tokens)
	return accAddr
}

func (s *IntegrationTestSuite) TestStakeAnteCreateValidatorThenDelegateSameTx() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	s.createValidatorAccs([]uint64{100})
	s.setStakeTrackerToCurrentBonded()

	amount := s.Setup.Stakingkeeper.TokensFromConsensusPower(s.Setup.Ctx, 1)
	accAddr, valAddr, privKey := s.newValidatorAccount(10_000, amount.MulRaw(3))
	createMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		privKey.PubKey(),
		sdk.NewCoin(s.Setup.Denom, amount),
		stakingtypes.Description{Moniker: "pending-validator"},
		stakingtypes.CommissionRates{
			Rate:          math.LegacyNewDecWithPrec(5, 1),
			MaxRate:       math.LegacyNewDecWithPrec(5, 1),
			MaxChangeRate: math.LegacyZeroDec(),
		},
		math.OneInt(),
	)
	s.Require().NoError(err)
	delegateMsg := stakingtypes.NewMsgDelegate(
		accAddr.String(),
		valAddr.String(),
		sdk.NewCoin(s.Setup.Denom, amount),
	)

	s.Require().NoError(s.runStakeAnte(s.T(), createMsg, delegateMsg))
	cacheCtx, err := s.runStakingMsgsOnCache(createMsg, delegateMsg)
	s.Require().NoError(err)
	validator, err := s.Setup.Stakingkeeper.GetValidator(cacheCtx, valAddr)
	s.Require().NoError(err)
	s.Require().Equal(amount.MulRaw(2), validator.Tokens)
}

func (s *IntegrationTestSuite) TestStakeAnteFivePercentUsesFinalTxState() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	totalBonded := s.setStakeTrackerToCurrentBonded()

	amount := totalBonded.QuoRaw(20).AddRaw(1)
	bob := s.newAccount(10_001, amount.MulRaw(2))
	stakingMsg := stakingtypes.NewMsgDelegate(bob.String(), valAddrs[0].String(), sdk.NewCoin(s.Setup.Denom, amount.MulRaw(2)))
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, err := stakingServer.Delegate(s.Setup.Ctx, stakingMsg)
	s.Require().NoError(err)
	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.Require().NoError(err)
	totalBonded = s.setStakeTrackerToCurrentBonded()

	alice := s.newAccount(10_002, amount)
	delegateMsg := stakingtypes.NewMsgDelegate(alice.String(), valAddrs[0].String(), sdk.NewCoin(s.Setup.Denom, amount))
	undelegateMsg := stakingtypes.NewMsgUndelegate(bob.String(), valAddrs[0].String(), sdk.NewCoin(s.Setup.Denom, amount))

	s.Require().NoError(s.runStakeAnte(s.T(), delegateMsg, undelegateMsg))
	cacheCtx, err := s.runStakingMsgsOnCache(delegateMsg, undelegateMsg)
	s.Require().NoError(err)
	afterTotal, err := s.Setup.Stakingkeeper.TotalBondedTokens(cacheCtx)
	s.Require().NoError(err)
	s.Require().Equal(totalBonded, afterTotal)
}

func (s *IntegrationTestSuite) TestStakeAnteRejectsStateStakingWouldMakeOverShareCap() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1)
	_, valAddrs, _ := s.createValidatorAccs([]uint64{100})
	totalBonded := s.setStakeTrackerToCurrentBonded()

	attacker := s.newAccount(10_003, totalBonded)
	initialStake := totalBonded.MulRaw(299).QuoRaw(701)
	stakingServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	_, err := stakingServer.Delegate(
		s.Setup.Ctx,
		stakingtypes.NewMsgDelegate(attacker.String(), valAddrs[0].String(), sdk.NewCoin(s.Setup.Denom, initialStake)),
	)
	s.Require().NoError(err)
	_, err = s.Setup.Stakingkeeper.EndBlocker(s.Setup.Ctx)
	s.Require().NoError(err)

	totalBonded = s.setStakeTrackerToCurrentBonded()
	attackerBonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(s.Setup.Ctx, attacker)
	s.Require().NoError(err)
	s.Require().False(attackerBonded.MulRaw(10).GT(totalBonded.MulRaw(3)))

	amount := totalBonded.MulRaw(3).Sub(attackerBonded.MulRaw(10)).QuoRaw(7).AddRaw(1)
	s.Require().True(amount.LT(totalBonded.QuoRaw(20)), "test must isolate share cap below the 5%% stake-change limit")
	s.Setup.MintTokens(attacker, amount)
	delegateMsg := stakingtypes.NewMsgDelegate(attacker.String(), valAddrs[0].String(), sdk.NewCoin(s.Setup.Denom, amount))

	s.Require().ErrorIs(s.runStakeAnte(s.T(), delegateMsg), reportertypes.ErrExceedsMaxStakeShare)
	cacheCtx, err := s.runStakingMsgsOnCache(delegateMsg)
	s.Require().NoError(err)
	afterTotal, err := s.Setup.Stakingkeeper.TotalBondedTokens(cacheCtx)
	s.Require().NoError(err)
	afterAttackerBonded, err := s.Setup.Stakingkeeper.GetDelegatorBonded(cacheCtx, attacker)
	s.Require().NoError(err)
	s.Require().True(afterAttackerBonded.MulRaw(10).GT(afterTotal.MulRaw(3)))
}
