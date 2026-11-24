package integration_test

import (
	"github.com/tellor-io/layer/testutil/sample"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// atomicFeeStakeFixture uses a real staking keeper record, but installs the
// deliberately fractional exchange rate needed to reproduce D1's 3-loya
// unrealizable split.  The validator is created through the integration
// setup first so its indexes and module-account state are genuine.
type atomicFeeStakeFixture struct {
	reporter  sdk.AccAddress
	selectors []sdk.AccAddress
	validator sdk.ValAddress
}

func (s *IntegrationTestSuite) atomicPrepareFeeStakeContext() {
	s.Setup.Ctx = s.Setup.Ctx.WithBlockHeight(1).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())
}

func (s *IntegrationTestSuite) atomicInstallStakeFixture(
	tokens math.Int,
	delegatorShares math.LegacyDec,
	ownerShares math.LegacyDec,
	selectorShares math.LegacyDec,
	selectorCount int,
) atomicFeeStakeFixture {
	s.atomicPrepareFeeStakeContext()
	owners, validators, _ := s.Setup.CreateValidators(1)
	owner, validatorAddr := owners[0], validators[0]
	reporter := sample.AccAddressBytes()
	selectors := make([]sdk.AccAddress, selectorCount)
	stakingMsgServer := stakingkeeper.NewMsgServerImpl(s.Setup.Stakingkeeper)
	for i := range selectors {
		selectors[i] = sample.AccAddressBytes()
		s.Setup.Accountkeeper.NewAccountWithAddress(s.Setup.Ctx, selectors[i])
		s.Setup.MintTokens(selectors[i], math.OneInt())
		s.Require().NoError(s.Setup.Reporterkeeper.Selectors.Set(
			s.Setup.Ctx,
			selectors[i],
			reportertypes.NewSelection(reporter, 1),
		))
		// Create the delegation through the SDK message server first.  This
		// initializes the distribution module's delegation starting-info and
		// keeps Unbond's real staking hooks available in the fee transaction.
		_, err := stakingMsgServer.Delegate(s.Setup.Ctx, stakingtypes.NewMsgDelegate(
			selectors[i].String(),
			validatorAddr.String(),
			sdk.NewCoin(s.Setup.Denom, math.OneInt()),
		))
		s.Require().NoError(err)
	}

	validator, err := s.Setup.Stakingkeeper.GetValidator(s.Setup.Ctx, validatorAddr)
	s.Require().NoError(err)
	// SetValidator does not update the power index itself.  Keep the real
	// validator indexed so the subsequent SDK Unbond calls exercise normal
	// staking state transitions.
	s.Require().NoError(s.Setup.Stakingkeeper.DeleteValidatorByPowerIndex(s.Setup.Ctx, validator))
	// Keep the bonded pool backed by exactly the tokens in the validator records.
	removed := validator.Tokens.Sub(tokens)
	s.Require().NoError(s.Setup.Bankkeeper.BurnCoins(s.Setup.Ctx, stakingtypes.BondedPoolName, sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, removed))))
	validator.Tokens = tokens
	validator.DelegatorShares = delegatorShares
	s.Require().NoError(s.Setup.Stakingkeeper.SetValidator(s.Setup.Ctx, validator))
	s.Require().NoError(s.Setup.Stakingkeeper.SetValidatorByPowerIndex(s.Setup.Ctx, validator))

	ownerDelegation, err := s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, owner, validatorAddr)
	s.Require().NoError(err)
	ownerDelegation.Shares = ownerShares
	s.Require().NoError(s.Setup.Stakingkeeper.SetDelegation(s.Setup.Ctx, ownerDelegation))

	for i := range selectors {
		selectorDelegation, getErr := s.Setup.Stakingkeeper.GetDelegation(s.Setup.Ctx, selectors[i], validatorAddr)
		s.Require().NoError(getErr)
		selectorDelegation.Shares = selectorShares
		s.Require().NoError(s.Setup.Stakingkeeper.SetDelegation(s.Setup.Ctx, selectorDelegation))
	}

	return atomicFeeStakeFixture{
		reporter:  reporter,
		selectors: selectors,
		validator: validatorAddr,
	}
}

func (s *IntegrationTestSuite) atomicExactSplitFixture() atomicFeeStakeFixture {
	return s.atomicInstallStakeFixture(
		math.NewInt(3),
		math.LegacyMustNewDecFromStr("4.800000000000000002"),
		math.LegacyMustNewDecFromStr("1.600000000000000002"),
		math.LegacyMustNewDecFromStr("1.6"),
		2,
	)
}

func (s *IntegrationTestSuite) atomicTwoExactPositionsFixture() atomicFeeStakeFixture {
	return s.atomicInstallStakeFixture(
		math.NewInt(4),
		math.LegacyNewDec(4),
		math.LegacyNewDec(2),
		math.LegacyNewDec(1),
		2,
	)
}

// atomicFeeFromStakeTx models the SDK transaction boundary.  The keeper must
// be allowed to mutate the cache and all mutations must be discarded on any
// returned error; only successful payment commits the cache.
func (s *IntegrationTestSuite) atomicFeeFromStakeTx(
	fixture atomicFeeStakeFixture,
	fee math.Int,
	hashID []byte,
) (sdk.Events, error) {
	cacheCtx, write := s.Setup.Ctx.CacheContext()
	err := s.Setup.Reporterkeeper.FeefromReporterStake(cacheCtx, fixture.reporter, fee, hashID, true)
	events := cacheCtx.EventManager().Events()
	if err == nil {
		write()
	}
	return events, err
}

func atomicDebitAmounts(events sdk.Events) []math.Int {
	amounts := make([]math.Int, 0)
	for _, event := range events {
		if event.Type != "deducted_stake_in_dispute" {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key != "amount" {
				continue
			}
			amount, ok := math.NewIntFromString(attr.Value)
			if ok {
				amounts = append(amounts, amount)
			}
			break
		}
	}
	return amounts
}

func (s *IntegrationTestSuite) atomicModuleBalance(module string) math.Int {
	addr := s.Setup.Accountkeeper.GetModuleAddress(module)
	return s.Setup.Bankkeeper.GetBalance(s.Setup.Ctx, addr, s.Setup.Denom).Amount
}

type atomicFeeStakeState struct {
	validator   stakingtypes.Validator
	delegations []stakingtypes.Delegation
	bonded      math.Int
	dispute     math.Int
	hasTracker  bool
	tracker     reportertypes.DelegationsAmounts
}

func (s *IntegrationTestSuite) atomicSnapshot(
	fixture atomicFeeStakeFixture,
	hashID []byte,
) atomicFeeStakeState {
	ctx := s.Setup.Ctx
	validator, err := s.Setup.Stakingkeeper.GetValidator(ctx, fixture.validator)
	s.Require().NoError(err)
	delegations := make([]stakingtypes.Delegation, len(fixture.selectors))
	for i, selector := range fixture.selectors {
		delegations[i], err = s.Setup.Stakingkeeper.GetDelegation(ctx, selector, fixture.validator)
		s.Require().NoError(err)
	}
	hasTracker, err := s.Setup.Reporterkeeper.HasFeePaidFromStakeByPayer(ctx, hashID, fixture.reporter)
	s.Require().NoError(err)
	var tracker reportertypes.DelegationsAmounts
	if hasTracker {
		tracker, err = s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(ctx, hashID, fixture.reporter)
		s.Require().NoError(err)
	}
	return atomicFeeStakeState{
		validator:   validator,
		delegations: delegations,
		bonded:      s.atomicModuleBalance(stakingtypes.BondedPoolName),
		dispute:     s.atomicModuleBalance(disputetypes.ModuleName),
		hasTracker:  hasTracker,
		tracker:     tracker,
	}
}

func (s *IntegrationTestSuite) atomicAssertUnchanged(
	before atomicFeeStakeState,
	fixture atomicFeeStakeFixture,
	hashID []byte,
) {
	s.Equal(before, s.atomicSnapshot(fixture, hashID))
}

func (s *IntegrationTestSuite) TestFeeFromReporterStakeRejectsUnrealizableThreeLoyaSplitAtomically() {
	fixture := s.atomicExactSplitFixture()
	hashID := []byte("d1-three-loya-unrealizable")
	before := s.atomicSnapshot(fixture, hashID)

	events, err := s.atomicFeeFromStakeTx(fixture, math.NewInt(2), hashID)
	s.Require().Error(err, "the split has independent floor capacity 1 + 1, but the second live debit is not realizable")
	s.ErrorContains(err, "short by 1")
	debits := atomicDebitAmounts(events)
	s.Require().Len(debits, 1, "the first debit must occur in the discarded SDK cache")
	s.Equal(math.OneInt(), debits[0])
	s.atomicAssertUnchanged(before, fixture, hashID)
}

func (s *IntegrationTestSuite) TestFeeFromReporterStakeRejectsAboveIndependentFloorCapacityAtomically() {
	fixture := s.atomicExactSplitFixture()
	hashID := []byte("d1-above-independent-floor")
	before := s.atomicSnapshot(fixture, hashID)

	// Each selector independently floors to one loya.  The requested three
	// loyas therefore exceeds the sum of independent floor capacities.
	events, err := s.atomicFeeFromStakeTx(fixture, math.NewInt(3), hashID)
	s.ErrorContains(err, "insufficient stake")
	s.Empty(atomicDebitAmounts(events), "above-cap admission must fail before any Unbond")
	s.atomicAssertUnchanged(before, fixture, hashID)
}

func (s *IntegrationTestSuite) TestFeeFromReporterStakeRollsBackAfterFirstDebitWhenBankTransferFails() {
	fixture := s.atomicTwoExactPositionsFixture()
	ctx := s.Setup.Ctx
	hashID := []byte("d1-late-bank-rollback")

	// Leave only one loya in the bonded module account while the real staking
	// records still expose two exact loya positions.  The first debit can
	// mutate staking state, but the final module-to-module transfer fails.
	bondedBeforeDrain := s.atomicModuleBalance(stakingtypes.BondedPoolName)
	sink := sample.AccAddressBytes()
	s.Setup.Accountkeeper.NewAccountWithAddress(ctx, sink)
	s.Require().NoError(s.Setup.Bankkeeper.SendCoinsFromModuleToAccount(
		ctx,
		stakingtypes.BondedPoolName,
		sink,
		sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, bondedBeforeDrain.Sub(math.OneInt()))),
	))
	s.Equal(math.OneInt(), s.atomicModuleBalance(stakingtypes.BondedPoolName))

	before := s.atomicSnapshot(fixture, hashID)

	events, err := s.atomicFeeFromStakeTx(fixture, math.NewInt(2), hashID)
	s.ErrorContains(err, "insufficient funds", "the bank transfer must fail after the first exact debit")
	debits := atomicDebitAmounts(events)
	s.Require().Len(debits, 2, "both exact debits occur in the discarded SDK cache before the bank failure")
	s.Equal(math.OneInt(), debits[0])
	s.Equal(math.OneInt(), debits[1])

	s.atomicAssertUnchanged(before, fixture, hashID)
}

func (s *IntegrationTestSuite) TestBondedFeeTopUpAddsOrigins() {
	fixture := s.atomicInstallStakeFixture(
		math.NewInt(8), math.LegacyNewDec(8), math.LegacyNewDec(2), math.LegacyNewDec(2), 3,
	)
	hashID := []byte("top-up-adds-origins")
	poolBefore := s.atomicModuleBalance(stakingtypes.BondedPoolName)
	disputeBefore := s.atomicModuleBalance(disputetypes.ModuleName)

	// Each selector starts with two loya. The first payment uses one selector.
	_, err := s.atomicFeeFromStakeTx(fixture, math.OneInt(), hashID)
	s.Require().NoError(err)
	first, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, hashID, fixture.reporter)
	s.Require().NoError(err)
	s.Require().Len(first.TokenOrigins, 1)
	s.Equal(math.OneInt(), first.Total)
	s.Equal(math.OneInt(), first.TokenOrigins[0].Amount)

	// The four-loya top-up debits the first selector again and reaches both others.
	events, err := s.atomicFeeFromStakeTx(fixture, math.NewInt(4), hashID)
	s.Require().NoError(err)
	s.ElementsMatch([]math.Int{math.OneInt(), math.OneInt(), math.NewInt(2)}, atomicDebitAmounts(events))
	merged, err := s.Setup.Reporterkeeper.FeePaidFromStakeByPayer(s.Setup.Ctx, hashID, fixture.reporter)
	s.Require().NoError(err)
	s.Require().Len(merged.TokenOrigins, 3)
	s.Equal(math.NewInt(5), merged.Total)
	var owners []sdk.AccAddress
	var amounts []math.Int
	firstOwner := sdk.AccAddress(first.TokenOrigins[0].DelegatorAddress)
	for _, origin := range merged.TokenOrigins {
		owner := sdk.AccAddress(origin.DelegatorAddress)
		owners = append(owners, owner)
		amounts = append(amounts, origin.Amount)
		s.Equal(fixture.validator.Bytes(), origin.ValidatorAddress)
		if firstOwner.Equals(owner) {
			s.Equal(math.NewInt(2), origin.Amount)
		}
	}
	s.ElementsMatch(fixture.selectors, owners)
	s.ElementsMatch([]math.Int{math.OneInt(), math.NewInt(2), math.NewInt(2)}, amounts)
	s.Equal(poolBefore.Sub(merged.Total), s.atomicModuleBalance(stakingtypes.BondedPoolName))
	s.Equal(disputeBefore.Add(merged.Total), s.atomicModuleBalance(disputetypes.ModuleName))
}
