package app_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/app"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	evidencetypes "cosmossdk.io/x/evidence/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	chainID       = "layertest-1"
	genesisFile   = "./testutils/sim-genesis.json"
	SimAppChainID = "simulation-app"
)

var FlagEnableStreamingValue bool

type storeKeysPrefixes struct {
	A        storetypes.StoreKey
	B        storetypes.StoreKey
	Prefixes [][]byte
}

// Get flags every time the simulator is run
func init() {
	simcli.GetSimulatorFlags()
	flag.BoolVar(&FlagEnableStreamingValue, "EnableStreaming", false, "Enable streaming service")
	sdk.DefaultBondDenom = "loya"
}

// fauxMerkleModeOpt returns a BaseApp option to use a dbStoreAdapter instead of
// an IAVLStore for faster simulation speed.
func fauxMerkleModeOpt(bapp *baseapp.BaseApp) {
	bapp.SetFauxMerkleMode()
}

// BenchmarkSimulation run the chain simulation
// Running using starport command:
// `starport chain simulate -v --numBlocks 200 --blockSize 50`
// Running as go benchmark test:
// `go test -benchmem -run=^$ -bench ^BenchmarkSimulation ./app -NumBlocks=200 -BlockSize 50 -Commit=true -Verbose=true -Enabled=true`
func BenchmarkSimulation(b *testing.B) {
	simcli.FlagSeedValue = time.Now().Unix()
	simcli.FlagVerboseValue = true
	simcli.FlagCommitValue = true
	simcli.FlagEnabledValue = true

	config := simcli.NewConfigFromFlags()
	config.ChainID = chainID
	config.GenesisFile = genesisFile

	db, dir, logger, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-bApp-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(b, err, "simulation setup failed")

	b.Cleanup(func() {
		require.NoError(b, db.Close())
		require.NoError(b, os.RemoveAll(dir))
	})

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = app.DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	bApp := app.New(
		logger,
		db,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(b, app.Name, bApp.Name())

	genesisJSON, err := os.ReadFile("./testutils/sim-genesis.json")
	require.NoError(b, err)

	var genesisMap map[string]json.RawMessage
	err = json.Unmarshal(genesisJSON, &genesisMap)
	require.NoError(b, err)

	// valSet, err := simtestutil.CreateRandomValidatorSet()
	// require.NoError(b, err)
	// fmt.Println("valSet:", valSet)

	// appState, _, err := simtestutil.AppStateFromGenesisFileFn(
	// 	rand.New(rand.NewSource(simcli.FlagSeedValue)),
	// 	bApp.AppCodec(),
	// 	"./testutils/sim-genesis.json",
	// )
	// fmt.Println("appState:", appState)
	// require.NoError(b, err)

	// randomAccounts := simtypes.RandomAccounts
	// fmt.Println("\nrandomAccounts:", randomAccounts(rand.New(rand.NewSource(simcli.FlagSeedValue)), b.N))
	// fmt.Println("\nb.N:", b.N)
	// fmt.Println("\nsimcli.FlagSeedValue: ", simcli.FlagSeedValue)

	// run randomized simulation
	_, simParams, _ := simulation.SimulateFromSeed(
		b,
		os.Stdout,
		bApp.BaseApp,
		simtestutil.AppStateFn(
			bApp.AppCodec(),
			bApp.SimulationManager(),
			genesisMap,
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(bApp, bApp.AppCodec(), config),
		bApp.ModuleAccountAddrs(),
		config,
		bApp.AppCodec(),
	)

	require.NotNil(b, bApp.MintKeeper)
	require.NotNil(b, bApp.MintKeeper.Minter)
	// fmt.Println("bApp.MintKeeper.Minter: ", bApp.MintKeeper.Minter)

	// export state and simParams before the simulation error is checked
	err = simtestutil.CheckExportSimulation(bApp, config, simParams)
	fmt.Println("err", err)
	// // require.NoError(b, err)
	// fmt.Println("simErr", simErr)
	// require.NoError(b, simErr)

	if config.Commit {
		simtestutil.PrintStats(db)
	}
}

// func TestAppStateDeterminism(t *testing.T) {
// 	// if !simcli.FlagEnabledValue {
// 	// 	t.Skip("skipping application simulation")
// 	// }

// 	config := simcli.NewConfigFromFlags()
// 	config.InitialBlockHeight = 1
// 	config.ExportParamsPath = ""
// 	config.OnOperation = true
// 	config.AllInvariants = true
// 	// config.GenesisFile = genesisFile

// 	var (
// 		r                    = rand.New(rand.NewSource(time.Now().Unix()))
// 		numSeeds             = 3
// 		numTimesToRunPerSeed = 5
// 		appHashList          = make([]json.RawMessage, numTimesToRunPerSeed)
// 		appOptions           = make(simtestutil.AppOptionsMap, 0)
// 	)
// 	appOptions[flags.FlagHome] = app.DefaultNodeHome
// 	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

// 	for i := 0; i < numSeeds; i++ {
// 		config.Seed = r.Int63()

// 		for j := 0; j < numTimesToRunPerSeed; j++ {
// 			var logger log.Logger
// 			if simcli.FlagVerboseValue {
// 				logger = log.NewTestLogger(t)
// 			} else {
// 				logger = log.NewNopLogger()
// 			}
// 			chainID := fmt.Sprintf("chain-id-%d-%d", i, j)
// 			config.ChainID = chainID

// 			db := dbm.NewMemDB()
// 			bApp := app.New(
// 				logger,
// 				db,
// 				nil,
// 				true,
// 				appOptions,
// 				fauxMerkleModeOpt,
// 				baseapp.SetChainID(chainID),
// 			)

// 			fmt.Printf(
// 				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
// 				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
// 			)

// 			genesisJSON, err := os.ReadFile("./testutils/sim-genesis.json")
// 			require.NoError(t, err)
// 			var genesisMap map[string]json.RawMessage
// 			err = json.Unmarshal(genesisJSON, &genesisMap)
// 			require.NoError(t, err)

// 			_, _, err = simulation.SimulateFromSeed(
// 				t,
// 				os.Stdout,
// 				bApp.BaseApp,
// 				simtestutil.AppStateFn(
// 					bApp.AppCodec(),
// 					bApp.SimulationManager(),
// 					genesisMap,
// 				),
// 				simtypes.RandomAccounts,
// 				simtestutil.SimulationOperations(bApp, bApp.AppCodec(), config),
// 				bApp.ModuleAccountAddrs(),
// 				config,
// 				bApp.AppCodec(),
// 			)
// 			require.NoError(t, err)

// 			if config.Commit {
// 				simtestutil.PrintStats(db)
// 			}

// 			appHash := bApp.LastCommitID().Hash
// 			appHashList[j] = appHash

// 			if j != 0 {
// 				require.Equal(
// 					t, string(appHashList[0]), string(appHashList[j]),
// 					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n", config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
// 				)
// 			}
// 		}
// 	}
// }

func TestAppImportExport(t *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = "layertest-1"
	config.GenesisFile = genesisFile

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application import/export simulation")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = app.DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	bApp := app.New(
		logger,
		db,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, app.Name, bApp.Name())
	// anteHandler := bApp.BaseApp.AnteHandler()
	// fmt.Println("\nanteHandler:", anteHandler)
	// simManager := bApp.SimulationManager()
	// fmt.Println("\nsimManager:", simManager)
	// weightedOps := simManager.WeightedOperations(module.SimulationState{})
	// // fmt.Println("\nweightedOps:", weightedOps)

	genesisJSON, err := os.ReadFile("./testutils/sim-genesis.json")
	require.NoError(t, err)

	var genesisMap map[string]json.RawMessage
	err = json.Unmarshal(genesisJSON, &genesisMap)
	require.NoError(t, err)

	// valSet, err := simtestutil.CreateRandomValidatorSet()
	// require.NoError(t, err)
	// fmt.Println("valSet:", valSet)

	// simtestutil.SetupWithConfiguration()

	// run randomized simulation
	_, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		bApp.BaseApp,
		simtestutil.AppStateFn(
			bApp.AppCodec(),
			bApp.SimulationManager(),
			genesisMap,
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(bApp, bApp.AppCodec(), config),
		bApp.BlockedModuleAccountAddrs(),
		config,
		bApp.AppCodec(),
	)
	require.NoError(t, simErr)

	// export state and simParams before the simulation error is checked
	err = simtestutil.CheckExportSimulation(bApp, config, simParams)
	require.NoError(t, err)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	fmt.Printf("exporting genesis...\n")

	exported, err := bApp.ExportAppStateAndValidators(false, []string{}, []string{})
	require.NoError(t, err)

	fmt.Printf("importing genesis...\n")

	newDB, newDir, _, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim-2",
		"Simulation-2",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	}()

	newApp := app.New(
		log.NewNopLogger(),
		newDB,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, app.Name, bApp.Name())

	var genesisState app.GenesisState
	err = json.Unmarshal(exported.AppState, &genesisState)
	require.NoError(t, err)

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Sprintf("%v", r)
			if !strings.Contains(err, "validator set is empty after InitGenesis") {
				panic(r)
			}
			logger.Info("Skipping simulation as all validators have been unbonded")
			logger.Info("err", err, "stacktrace", string(debug.Stack()))
		}
	}()

	ctxA := bApp.NewContextLegacy(true, tmproto.Header{Height: bApp.LastBlockHeight()})
	ctxB := newApp.NewContextLegacy(true, tmproto.Header{Height: bApp.LastBlockHeight()})
	_, err = newApp.ModuleManager().InitGenesis(ctxB, bApp.AppCodec(), genesisState)
	require.NoError(t, err)
	require.NoError(t, newApp.StoreConsensusParams(ctxB, exported.ConsensusParams))

	fmt.Printf("comparing stores...\n")

	storeKeysPrefixes := []storeKeysPrefixes{
		{bApp.GetKey(authtypes.StoreKey), newApp.GetKey(authtypes.StoreKey), [][]byte{}},
		{
			bApp.GetKey(stakingtypes.StoreKey), newApp.GetKey(stakingtypes.StoreKey),
			[][]byte{
				stakingtypes.UnbondingQueueKey, stakingtypes.RedelegationQueueKey, stakingtypes.ValidatorQueueKey,
				stakingtypes.HistoricalInfoKey, stakingtypes.UnbondingIDKey, stakingtypes.UnbondingIndexKey, stakingtypes.UnbondingTypeKey, stakingtypes.ValidatorUpdatesKey,
			},
		}, // ordering may change but it doesn't matter
		{bApp.GetKey(slashingtypes.StoreKey), newApp.GetKey(slashingtypes.StoreKey), [][]byte{}},
		{bApp.GetKey(minttypes.StoreKey), newApp.GetKey(minttypes.StoreKey), [][]byte{}},
		{bApp.GetKey(distrtypes.StoreKey), newApp.GetKey(distrtypes.StoreKey), [][]byte{}},
		{bApp.GetKey(banktypes.StoreKey), newApp.GetKey(banktypes.StoreKey), [][]byte{banktypes.BalancesPrefix}},
		{bApp.GetKey(paramstypes.StoreKey), newApp.GetKey(paramstypes.StoreKey), [][]byte{}},
		{bApp.GetKey(govtypes.StoreKey), newApp.GetKey(govtypes.StoreKey), [][]byte{}},
		{bApp.GetKey(evidencetypes.StoreKey), newApp.GetKey(evidencetypes.StoreKey), [][]byte{}},
		{bApp.GetKey(capabilitytypes.StoreKey), newApp.GetKey(capabilitytypes.StoreKey), [][]byte{}},
		{bApp.GetKey(authzkeeper.StoreKey), newApp.GetKey(authzkeeper.StoreKey), [][]byte{authzkeeper.GrantKey, authzkeeper.GrantQueuePrefix}},
	}

	for _, skp := range storeKeysPrefixes {
		storeA := ctxA.KVStore(skp.A)
		storeB := ctxB.KVStore(skp.B)

		failedKVAs, failedKVBs := simtestutil.DiffKVStores(storeA, storeB, skp.Prefixes)
		require.Equal(t, len(failedKVAs), len(failedKVBs), "unequal sets of key-values to compare")

		fmt.Printf("compared %d different key/value pairs between %s and %s\n", len(failedKVAs), skp.A, skp.B)
		require.Equal(t, 0, len(failedKVAs), simtestutil.GetSimulationLog(skp.A.Name(), bApp.SimulationManager().StoreDecoders, failedKVAs, failedKVBs))
	}
}

func TestAppSimulationAfterImport(t *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = "mars-simapp-after-import"

	db, dir, logger, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application simulation after import")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = app.DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	bApp := app.New(
		logger,
		db,
		nil,
		true,
		appOptions,
		fauxMerkleModeOpt,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, app.Name, bApp.Name())

	// run randomized simulation
	stopEarly, simParams, simErr := simulation.SimulateFromSeed(
		t,
		os.Stdout,
		bApp.BaseApp,
		simtestutil.AppStateFn(
			bApp.AppCodec(),
			bApp.SimulationManager(),
			bApp.BasicModuleManager.DefaultGenesis(bApp.AppCodec()),
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(bApp, bApp.AppCodec(), config),
		bApp.BlockedModuleAccountAddrs(),
		config,
		bApp.AppCodec(),
	)
	require.NoError(t, simErr)

	// export state and simParams before the simulation error is checked
	err = simtestutil.CheckExportSimulation(bApp, config, simParams)
	require.NoError(t, err)

	if config.Commit {
		simtestutil.PrintStats(db)
	}

	if stopEarly {
		fmt.Println("can't export or import a zero-validator genesis, exiting test...")
		return
	}

	fmt.Printf("exporting genesis...\n")

	exported, err := bApp.ExportAppStateAndValidators(true, []string{}, []string{})
	require.NoError(t, err)

	fmt.Printf("importing genesis...\n")

	newDB, newDir, _, _, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim-2",
		"Simulation-2",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, newDB.Close())
		require.NoError(t, os.RemoveAll(newDir))
	}()

	newApp := app.New(
		log.NewNopLogger(),
		newDB,
		nil,
		true,
		appOptions,
		fauxMerkleModeOpt,
		baseapp.SetChainID(config.ChainID),
	)
	require.Equal(t, app.Name, bApp.Name())

	_, err = newApp.InitChain(&abci.RequestInitChain{
		ChainId:       config.ChainID,
		AppStateBytes: exported.AppState,
	})
	require.NoError(t, err)
	_, _, err = simulation.SimulateFromSeed(
		t,
		os.Stdout,
		newApp.BaseApp,
		simtestutil.AppStateFn(
			bApp.AppCodec(),
			bApp.SimulationManager(),
			bApp.BasicModuleManager.DefaultGenesis(bApp.AppCodec()),
		),
		simtypes.RandomAccounts,
		simtestutil.SimulationOperations(newApp, newApp.AppCodec(), config),
		newApp.BlockedModuleAccountAddrs(),
		config,
		bApp.AppCodec(),
	)
	require.NoError(t, err)
}

func TestFullAppSimulation(t *testing.T) {
	config := simcli.NewConfigFromFlags()
	config.ChainID = chainID

	db, dir, _, skip, err := simtestutil.SetupSimulation(
		config,
		"leveldb-app-sim",
		"Simulation",
		simcli.FlagVerboseValue,
		simcli.FlagEnabledValue,
	)
	if skip {
		t.Skip("skipping application simulation")
	}
	require.NoError(t, err, "simulation setup failed")

	defer func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.RemoveAll(dir))
	}()

	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = app.DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	// app := NewSimApp(logger, db, nil, true, appOptions, fauxMerkleModeOpt, baseapp.SetChainID(config.ChainID))
}

func TestAppStateDeterminism2(t *testing.T) {
	if !simcli.FlagEnabledValue {
		t.Skip("skipping application simulation")
	}

	config := simcli.NewConfigFromFlags()
	config.InitialBlockHeight = 1
	config.ExportParamsPath = ""
	config.OnOperation = false
	config.AllInvariants = false
	config.ChainID = SimAppChainID

	numSeeds := 3
	numTimesToRunPerSeed := 3 // This used to be set to 5, but we've temporarily reduced it to 3 for the sake of faster CI.
	appHashList := make([]json.RawMessage, numTimesToRunPerSeed)

	// We will be overriding the random seed and just run a single simulation on the provided seed value
	if config.Seed != simcli.DefaultSeedValue {
		numSeeds = 1
	}

	appOptions := viper.New()
	if FlagEnableStreamingValue {
		m := make(map[string]interface{})
		m["streaming.abci.keys"] = []string{"*"}
		m["streaming.abci.plugin"] = "abci_v1"
		m["streaming.abci.stop-node-on-err"] = true
		for key, value := range m {
			appOptions.SetDefault(key, value)
		}
	}
	// appOptions.SetDefault(flags.FlagHome, DefaultNodeHome)
	appOptions.SetDefault(server.FlagInvCheckPeriod, simcli.FlagPeriodValue)
	if simcli.FlagVerboseValue {
		appOptions.SetDefault(flags.FlagLogLevel, "debug")
	}

	for i := 0; i < numSeeds; i++ {
		if config.Seed == simcli.DefaultSeedValue {
			config.Seed = rand.Int63()
		}

		fmt.Println("config.Seed: ", config.Seed)

		for j := 0; j < numTimesToRunPerSeed; j++ {
			var logger log.Logger
			if simcli.FlagVerboseValue {
				logger = log.NewTestLogger(t)
			} else {
				logger = log.NewNopLogger()
			}

			db := dbm.NewMemDB()
			app := app.New(logger, db, nil, true, appOptions, interBlockCacheOpt(), baseapp.SetChainID(SimAppChainID))
			if !simcli.FlagSigverifyTxValue {
				app.SetNotSigverifyTx()
			}

			fmt.Printf(
				"running non-determinism simulation; seed %d: %d/%d, attempt: %d/%d\n",
				config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
			)

			_, _, err := simulation.SimulateFromSeed(
				t,
				os.Stdout,
				app.BaseApp,
				simtestutil.AppStateFn(app.AppCodec(), app.SimulationManager(), app.DefaultGenesis()),
				simtypes.RandomAccounts, // Replace with own random account function if using keys other than secp256k1
				simtestutil.SimulationOperations(app, app.AppCodec(), config),
				BlockedAddresses(),
				config,
				app.AppCodec(),
			)
			require.NoError(t, err)

			if config.Commit {
				simtestutil.PrintStats(db)
			}

			appHash := app.LastCommitID().Hash
			appHashList[j] = appHash

			if j != 0 {
				require.Equal(
					t, string(appHashList[0]), string(appHashList[j]),
					"non-determinism in seed %d: %d/%d, attempt: %d/%d\n", config.Seed, i+1, numSeeds, j+1, numTimesToRunPerSeed,
				)
			}
		}
	}
}

// interBlockCacheOpt returns a BaseApp option function that sets the persistent
// inter-block write-through cache.
func interBlockCacheOpt() func(*baseapp.BaseApp) {
	return baseapp.SetInterBlockCache(store.NewCommitKVStoreCacheManager())
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range GetMaccPerms() {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	// allow the following addresses to receive funds
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dup := make(map[string][]string)
	for acc, perms := range app.MaccPerms {
		dup[acc] = perms
	}

	return dup
}
