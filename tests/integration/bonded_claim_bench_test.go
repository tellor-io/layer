package integration_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	setup "github.com/tellor-io/layer/tests"
	disputekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputetypes "github.com/tellor-io/layer/x/dispute/types"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BenchmarkBondedFeeClaim times the public SUPPORT claim and its cache write.
// An empty transaction cache is the matched control. Set LAYER_CLAIM_PAIRS
// to measure another tracker size. LAYER_CLAIM_VALIDATORS can select the six-
// validator Palmito fixture instead of the default 20-validator mainnet fixture.
func BenchmarkBondedFeeClaim(b *testing.B) {
	const (
		validators  = 20
		selectors   = 100
		delegations = 10
	)
	pairs := selectors * delegations
	if raw, ok := os.LookupEnv("LAYER_CLAIM_PAIRS"); ok {
		var err error
		pairs, err = strconv.Atoi(raw)
		require.NoError(b, err)
		require.GreaterOrEqual(b, pairs, 1)
		require.LessOrEqual(b, pairs, selectors*delegations)
	}
	s := new(setup.SharedSetup)
	s.SetupTest(b)
	ctx := s.Ctx.WithBlockHeight(2)
	_, valAddrs, _ := s.CreateValidators(validators - 1) // SharedSetup starts with one genesis validator.
	activeCount := validators
	if raw, ok := os.LookupEnv("LAYER_CLAIM_VALIDATORS"); ok {
		var err error
		activeCount, err = strconv.Atoi(raw)
		require.NoError(b, err)
		require.Contains(b, []int{6, validators}, activeCount)
	}

	// CreateValidators uses the first incremental account addresses. Use
	// disjoint owners here and give each ten live staking positions.
	owners := simtestutil.CreateIncrementalAccounts(validators + selectors + 1)[validators:]
	payer := owners[0]
	residualPayer := owners[selectors]
	weight := math.NewInt(1_000_000_000)
	origins := make([]*reportertypes.TokenOriginInfo, 0, pairs)
	staking := stakingkeeper.NewMsgServerImpl(s.Stakingkeeper)
	for _, owner := range owners[:selectors] {
		s.MintTokens(owner, weight.MulRaw(delegations))
		for _, val := range valAddrs[:delegations] {
			_, err := staking.Delegate(ctx, stakingtypes.NewMsgDelegate(owner.String(), val.String(), sdk.NewCoin(s.Denom, weight)))
			require.NoError(b, err)
			if len(origins) < pairs {
				origins = append(origins, &reportertypes.TokenOriginInfo{DelegatorAddress: owner, ValidatorAddress: val, Amount: weight})
			}
		}
		require.NoError(b, s.Reporterkeeper.Selectors.Set(ctx, owner, reportertypes.NewSelection(payer, delegations)))
	}
	if activeCount != validators {
		stakingParams, err := s.Stakingkeeper.GetParams(ctx)
		require.NoError(b, err)
		stakingParams.MaxValidators = uint32(activeCount)
		require.NoError(b, s.Stakingkeeper.SetParams(ctx, stakingParams))
		_, err = s.Stakingkeeper.EndBlocker(ctx)
		require.NoError(b, err)
	}
	active, err := s.Stakingkeeper.GetBondedValidatorsByPower(ctx)
	require.NoError(b, err)
	require.Len(b, active, activeCount)
	require.NoError(b, s.Reporterkeeper.Reporters.Set(ctx, payer, reportertypes.NewReporter(reportertypes.DefaultMinCommissionRate, math.OneInt(), "claim")))
	params, err := s.Reporterkeeper.Params.Get(ctx)
	require.NoError(b, err)
	params.MaxValidatorPowerShare = reportertypes.DefaultMaxValidatorPowerShare
	params.MaxReporterPowerShare = reportertypes.DefaultMaxReporterPowerShare
	require.NoError(b, s.Reporterkeeper.Params.Set(ctx, params))

	// The last origin is one loya smaller. A second payer supplied that
	// loya. For a quote divisible by 20 the partial refund therefore leaves
	// exactly pairs-1 whole loya for the allocator's repeated remainder scan.
	origins[pairs-1].Amount = weight.SubRaw(1)
	quote := weight.MulRaw(int64(pairs))
	gross := quote.SubRaw(1)
	refund, _ := disputekeeper.CalculateRefundAmount(gross, quote)
	floors := math.ZeroInt()
	for _, origin := range origins {
		floors = floors.Add(origin.Amount.Mul(refund).Quo(gross))
	}
	require.True(b, refund.Sub(floors).Equal(math.NewInt(int64(pairs-1))), "unexpected remainder count %s", refund.Sub(floors))

	const id = uint64(1)
	hash := []byte("bonded-claim")
	receipt := reportertypes.DelegationsAmounts{TokenOrigins: origins, Total: gross}
	require.NoError(b, s.Reporterkeeper.FeePaidFromStake.Set(ctx, feePaidFromStakePayerKey(hash, payer), receipt))
	require.NoError(b, s.Reporterkeeper.DisputedDelegationAmounts.Set(ctx, hash, reportertypes.DelegationsAmounts{
		Total: quote, TokenOrigins: []*reportertypes.TokenOriginInfo{{DelegatorAddress: sdk.AccAddress(valAddrs[0]), ValidatorAddress: valAddrs[0], Amount: quote}},
	}))
	burn := quote.QuoRaw(20)
	require.NoError(b, s.Disputekeeper.Disputes.Set(ctx, id, disputetypes.Dispute{
		DisputeId: id, HashId: hash, DisputeRound: 1, PrevDisputeIds: []uint64{id}, DisputeStatus: disputetypes.Resolved,
		DisputeCategory: disputetypes.Major, DisputeFee: quote, FeeTotal: quote, SlashAmount: quote, BurnAmount: burn,
	}))
	require.NoError(b, s.Disputekeeper.Votes.Set(ctx, id, disputetypes.Vote{Id: id, VoteResult: disputetypes.VoteResult_SUPPORT, Executed: true}))
	require.NoError(b, s.Disputekeeper.DisputeFeePayer.Set(ctx, collections.Join(id, payer.Bytes()), disputetypes.PayerInfo{Amount: gross, FromBond: true}))
	require.NoError(b, s.Disputekeeper.DisputeFeePayer.Set(ctx, collections.Join(id, residualPayer.Bytes()), disputetypes.PayerInfo{Amount: math.OneInt()}))
	require.NoError(b, s.Disputekeeper.Dust.Set(ctx, math.ZeroInt()))
	coins := sdk.NewCoins(sdk.NewCoin(s.Denom, quote.MulRaw(2).Sub(burn)))
	require.NoError(b, s.Bankkeeper.MintCoins(ctx, authtypes.Minter, coins))
	require.NoError(b, s.Bankkeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, disputetypes.ModuleName, coins))

	server := disputekeeper.NewMsgServerImpl(s.Disputekeeper)
	msg := &disputetypes.MsgWithdrawFeeRefund{Id: id, PayerAddress: payer.String(), CallerAddress: payer.String()}
	for _, name := range []string{"empty", "withdraw"} {
		b.Run(fmt.Sprintf("%s/%d-pairs/%d-validators", name, pairs, activeCount), func(b *testing.B) {
			var gas storetypes.Gas
			var parent sdk.Context
			for i := 0; i < b.N; i++ {
				parent, _ = ctx.CacheContext()
				tx, write := parent.WithGasMeter(storetypes.NewInfiniteGasMeter()).CacheContext()
				if name == "withdraw" {
					if _, err := server.WithdrawFeeRefund(tx, msg); err != nil {
						b.Fatal(err)
					}
				}
				write()
				gas = tx.GasMeter().GasConsumed()
			}
			b.StopTimer()
			b.ReportMetric(float64(gas), "gas/op")
			if name == "withdraw" {
				// Both claim legs hit the cap. All stake must have reached the
				// unbonding pool, including the final principal leg to the payer.
				pool := s.Accountkeeper.GetModuleAddress(stakingtypes.NotBondedPoolName)
				initial := s.Bankkeeper.GetBalance(ctx, pool, s.Denom).Amount
				final := s.Bankkeeper.GetBalance(parent, pool, s.Denom).Amount
				require.Equal(b, gross.Add(refund), final.Sub(initial))
				has, err := s.Reporterkeeper.HasFeePaidFromStakeByPayer(parent, hash, payer)
				require.NoError(b, err)
				require.False(b, has)
			}
		})
	}
}
