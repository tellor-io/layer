package fork_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdkStore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cosmosdb "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"
)

type ReporterStateEntry struct {
	ReporterAddr []byte                       `json:"reporter_addr"`
	Reporter     reportertypes.OracleReporter `json:"reporter"`
}

type SelectorStateEntry struct {
	SelectorAddr []byte                  `json:"selector_addr"`
	Selector     reportertypes.Selection `json:"selector"`
}

type SelectorTipsStateEntry struct {
	SelectorAddress []byte `json:"selector_address"`
	Tips            string `json:"tips"`
}

type DisputedDelegationAmountStateEntry struct {
	HashId           []byte                           `json:"hash_id"`
	DelegationAmount reportertypes.DelegationsAmounts `json:"delegation_amount"`
}

type FeePaidFromStakeStateEntry struct {
	HashId           []byte                           `json:"hash_id"`
	DelegationAmount reportertypes.DelegationsAmounts `json:"delegation_amount"`
}

type ModuleStateData struct {
	Reporters                 []ReporterStateEntry                 `json:"reporters"`
	Selectors                 []SelectorStateEntry                 `json:"selectors"`
	SelectorTips              []SelectorTipsStateEntry             `json:"selector_tips"`
	DisputedDelegationAmounts []DisputedDelegationAmountStateEntry `json:"disputed_delegation_amounts"`
	FeePaidFromStake          []FeePaidFromStakeStateEntry         `json:"fee_paid_from_stake"`
}

func setupTest(t *testing.T) (context.Context, store.KVStoreService, codec.Codec, keeper.Keeper) {
	t.Helper()
	// Create in-memory store
	storeKey := storetypes.NewKVStoreKey(reportertypes.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(reportertypes.MemStoreKey)
	db := cosmosdb.NewMemDB()

	stateStore := sdkStore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	storeService := runtime.NewKVStoreService(storeKey)
	cdc := codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())

	accountKeeper := new(mocks.AccountKeeper)
	bankKeeper := new(mocks.BankKeeper)
	stakingKeeper := new(mocks.StakingKeeper)
	registryKeeper := new(mocks.RegistryKeeper)
	k := keeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authtypes.NewModuleAddress(govtypes.ModuleName).String(), accountKeeper, stakingKeeper, bankKeeper, registryKeeper)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	return ctx, storeService, cdc, k
}

func TestMigrateFork(t *testing.T) {
	ctx, _, _, k := setupTest(t)

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	m := keeper.NewMigrator(k)

	err := m.MigrateFork(sdkCtx)
	require.NoError(t, err)

	// Read the test data file
	file, err := os.Open("reporter_module_state.json")
	require.NoError(t, err)
	defer file.Close()

	// Read the file contents
	fileData, err := io.ReadAll(file)
	require.NoError(t, err)

	// Parse the JSON data
	var moduleState ModuleStateData
	err = json.Unmarshal(fileData, &moduleState)
	require.NoError(t, err)

	bytesKeyCodec := collections.BytesKey
	// Verify reporters were migrated correctly
	for _, entry := range moduleState.Reporters {
		key := make([]byte, bytesKeyCodec.Size(entry.ReporterAddr))
		bytesKeyCodec.Encode(key, entry.ReporterAddr)
		fmt.Println("key: ", hex.EncodeToString(key))
		reporter, err := k.Reporters.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, entry.Reporter, reporter)
	}

	// Verify selectors were migrated correctly
	for _, entry := range moduleState.Selectors {
		key := make([]byte, bytesKeyCodec.Size(entry.SelectorAddr))
		bytesKeyCodec.Encode(key, entry.SelectorAddr)
		selector, err := k.Selectors.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, entry.Selector, selector)
	}

	// Verify selector tips were migrated correctly
	for _, entry := range moduleState.SelectorTips {
		key := make([]byte, bytesKeyCodec.Size(entry.SelectorAddress))
		bytesKeyCodec.Encode(key, entry.SelectorAddress)
		selectorTips, err := k.SelectorTips.Get(ctx, key)
		require.NoError(t, err)
		tips := math.LegacyMustNewDecFromStr(entry.Tips)
		require.Equal(t, tips, selectorTips)
	}

	// Verify disputed delegation amounts were migrated correctly
	for _, entry := range moduleState.DisputedDelegationAmounts {
		key := make([]byte, bytesKeyCodec.Size(entry.HashId))
		bytesKeyCodec.Encode(key, entry.HashId)
		disputedDelegationAmount, err := k.DisputedDelegationAmounts.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, entry.DelegationAmount, disputedDelegationAmount)
	}

	// Verify fee paid from stake were migrated correctly
	for _, entry := range moduleState.FeePaidFromStake {
		key := make([]byte, bytesKeyCodec.Size(entry.HashId))
		bytesKeyCodec.Encode(key, entry.HashId)
		feePaidFromStake, err := k.FeePaidFromStake.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, entry.DelegationAmount, feePaidFromStake)
	}
}
