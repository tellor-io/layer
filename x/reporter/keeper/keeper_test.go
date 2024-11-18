package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/sample"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/colltest"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupKeeper(tb testing.TB) (keeper.Keeper, *mocks.StakingKeeper, *mocks.BankKeeper, *mocks.RegistryKeeper, sdk.Context, store.KVStoreService) {
	tb.Helper()
	return keepertest.ReporterKeeper(tb)
}

func TestKeeper(t *testing.T) {
	k, sk, bk, _, ctx, _ := keepertest.ReporterKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
}

func TestGetAuthority(t *testing.T) {
	k, _, _, _, _, _ := setupKeeper(t)
	authority := k.GetAuthority()
	require.NotEmpty(t, authority)

	// convert to address and check if it's valid
	_, err := sdk.AccAddressFromBech32(authority)
	require.NoError(t, err)
}

func TestLogger(t *testing.T) {
	k, _, _, _, _, _ := setupKeeper(t)
	require.NotNil(t, k.Logger())
}

func TestGetDelegatorTokensAtBlock(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	delAddr, val1Address, val2Address := sample.AccAddressBytes(), sdk.ValAddress(sample.AccAddressBytes()), sdk.ValAddress(sample.AccAddressBytes())
	require.NoError(t, k.Selectors.Set(ctx, delAddr, types.NewSelection(delAddr, 2)))

	tokenOrigin1 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr,
		ValidatorAddress: val1Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	tokenOrigin2 := &types.TokenOriginInfo{
		DelegatorAddress: delAddr,
		ValidatorAddress: val2Address,
		Amount:           math.NewIntWithDecimal(1000, 6),
	}
	tokenOrigins := []*types.TokenOriginInfo{tokenOrigin1, tokenOrigin2}
	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(delAddr.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{TokenOrigins: tokenOrigins, Total: tokenOrigin1.Amount.Add(tokenOrigin2.Amount)}))

	tokens, err := k.GetDelegatorTokensAtBlock(ctx, delAddr, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.NewIntWithDecimal(2000, 6), tokens)
}

func TestGetReporterTokensAtBlock(t *testing.T) {
	k, _, _, _, ctx, _ := setupKeeper(t)
	reporter := sample.AccAddressBytes()
	tokens, err := k.GetReporterTokensAtBlock(ctx, reporter, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt(), tokens)

	require.NoError(t, k.Report.Set(ctx, collections.Join([]byte{}, collections.Join(reporter.Bytes(), uint64(ctx.BlockHeight()))), types.DelegationsAmounts{Total: math.OneInt()}))

	tokens, err = k.GetReporterTokensAtBlock(ctx, reporter, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), tokens)

	tokens, err = k.GetReporterTokensAtBlock(ctx, reporter, uint64(ctx.BlockHeight()+10))
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), tokens)
}

func TestTrackStakeChange(t *testing.T) {
	k, sk, _, _, ctx, _ := setupKeeper(t)
	expiration := ctx.BlockTime().Add(1)
	err := k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(t, err)
	require.NoError(t, k.TrackStakeChange(ctx))

	expiration = ctx.BlockTime()
	err = k.Tracker.Set(ctx, types.StakeTracker{Expiration: &expiration, Amount: math.NewInt(1000)})
	require.NoError(t, err)
	sk.On("TotalBondedTokens", ctx).Return(math.OneInt(), nil)
	require.NoError(t, k.TrackStakeChange(ctx))

	change, err := k.Tracker.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.OneInt(), change.Amount)
	require.Equal(t, expiration.Add(12*time.Hour), *change.Expiration)
}

type Range[K any] struct {
	start *collections.RangeKey[K]
	end   *collections.RangeKey[K]
	order collections.Order
}

func (r *Range[K]) RangeValues() (start, end *collections.RangeKey[K], order collections.Order, err error) {
	return r.start, r.end, r.order, nil
}

func NewSuperPrefixedTripleRange[K1, K2, K3 any](k1 K1, k2 K2) collections.Ranger[collections.Triple[K1, K2, K3]] {
	key := collections.TripleSuperPrefix[K1, K2, K3](k1, k2)
	return &Range[collections.Triple[K1, K2, K3]]{
		start: collections.RangeKeyExact(key),
		end:   collections.RangeKeyPrefixEnd(key),
	}
}

type ReporterBlockNumberIndexes struct {
	BlockNumber *indexes.ReversePair[[]byte, collections.Pair[string, uint64], uint64]
}

func newReportIndexes(sb *collections.SchemaBuilder) ReporterBlockNumberIndexes {
	return ReporterBlockNumberIndexes{
		BlockNumber: indexes.NewReversePair[uint64](
			sb, collections.NewPrefix("reporter_blocknumber"), "info_by_reporter_blocknumber_index",
			collections.PairKeyCodec(collections.BytesKey, collections.PairKeyCodec(collections.StringKey, collections.Uint64Key)),
			// indexes.WithReversePairUncheckedValue(), // denom to address indexes were stored as Key: Join(denom, address) Value: []byte{0}, this will migrate the value to []byte{} in a lazy way.
		),
	}
}

func (b ReporterBlockNumberIndexes) IndexesList() []collections.Index[collections.Pair[[]byte, collections.Pair[string, uint64]], uint64] {
	return []collections.Index[collections.Pair[[]byte, collections.Pair[string, uint64]], uint64]{b.BlockNumber}
}

func TestTripleRange(t *testing.T) {
	sk, ctx := colltest.MockStore()
	schema := collections.NewSchemaBuilder(sk)
	kc := collections.PairKeyCodec(collections.BytesKey, collections.PairKeyCodec(collections.StringKey, collections.Uint64Key))

	indexedMap := collections.NewIndexedMap(
		schema,
		collections.NewPrefix("reports"), "reports",
		kc,
		collections.Uint64Value,
		newReportIndexes(schema),
	)

	keys := []collections.Pair[[]byte, collections.Pair[string, uint64]]{
		collections.Join([]byte("queryid1"), collections.Join("reporterA", uint64(1))),
		collections.Join([]byte("queryid2"), collections.Join("reporterA", uint64(1))),
		collections.Join([]byte("queryid3"), collections.Join("reporterA", uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join("reporterB", uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join("reporterC", uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join("reporterD", uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join("reporterD", uint64(1))),
		collections.Join([]byte("queryid1"), collections.Join("reporterA", uint64(6))),
		collections.Join([]byte("queryid2"), collections.Join("reporterA", uint64(6))),
	}

	for _, k := range keys {
		require.NoError(t, indexedMap.Set(ctx, k, uint64(1)))
	}

	kg := collections.PairKeyCodec(collections.StringKey, collections.Uint64Key)
	startBuffer := make([]byte, kg.Size(collections.Join("reporterA", uint64(0))))
	endBuffer := make([]byte, kg.Size(collections.Join("reporterA", uint64(7))))
	_, err := kg.Encode(startBuffer, collections.Join("reporterA", uint64(0)))
	require.NoError(t, err)
	_, err = kg.Encode(endBuffer, collections.Join("reporterA", uint64(7)))
	require.NoError(t, err)

	iter, err := indexedMap.Indexes.BlockNumber.IterateRaw(ctx, startBuffer, endBuffer, collections.OrderDescending)
	require.NoError(t, err)
	gotKeys, err := iter.Keys()
	require.NoError(t, err)
	require.Equal(t, 5, len(gotKeys))
}
