package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	abci "github.com/cometbft/cometbft/abci/types"
	tmos "github.com/cometbft/cometbft/libs/os"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	icq "github.com/cosmos/ibc-apps/modules/async-icq/v8"
	icqkeeper "github.com/cosmos/ibc-apps/modules/async-icq/v8/keeper"
	icqtypes "github.com/cosmos/ibc-apps/modules/async-icq/v8/types"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcporttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/spf13/cast"
	globalfee "github.com/strangelove-ventures/globalfee/x/globalfee"
	globalfeekeeper "github.com/strangelove-ventures/globalfee/x/globalfee/keeper"
	globalfeetypes "github.com/strangelove-ventures/globalfee/x/globalfee/types"
	_ "github.com/tellor-io/layer/app/config"
	appflags "github.com/tellor-io/layer/app/flags"
	"github.com/tellor-io/layer/docs"
	bridgemodule "github.com/tellor-io/layer/x/bridge"
	bridgemodulekeeper "github.com/tellor-io/layer/x/bridge/keeper"
	bridgemoduletypes "github.com/tellor-io/layer/x/bridge/types"
	disputemodule "github.com/tellor-io/layer/x/dispute"
	disputemodulekeeper "github.com/tellor-io/layer/x/dispute/keeper"
	disputemoduletypes "github.com/tellor-io/layer/x/dispute/types"
	"github.com/tellor-io/layer/x/mint"
	mintkeeper "github.com/tellor-io/layer/x/mint/keeper"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	oraclemodule "github.com/tellor-io/layer/x/oracle"
	oraclemodulekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	oraclemoduletypes "github.com/tellor-io/layer/x/oracle/types"
	registrymodulekeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrymodule "github.com/tellor-io/layer/x/registry/module"
	registrymoduletypes "github.com/tellor-io/layer/x/registry/types"
	reportermodulekeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportermodule "github.com/tellor-io/layer/x/reporter/module"
	reportermoduletypes "github.com/tellor-io/layer/x/reporter/types"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sigtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	Name = "layer"
	// BondDenom defines the native staking token denomination.
	BondDenom = "loya"

	// DisplayDenom defines the name, symbol, and display value of the Tellor Tributes token.
	DisplayDenom = "TRB"
)

// this line is used by starport scaffolding # stargate/wasm/app/enabledProposals

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:         nil,
		distrtypes.ModuleName:              nil,
		icatypes.ModuleName:                nil,
		minttypes.TimeBasedRewards:         nil,
		minttypes.ModuleName:               {authtypes.Minter},
		stakingtypes.BondedPoolName:        {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:     {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:                {authtypes.Burner},
		ibctransfertypes.ModuleName:        {authtypes.Minter, authtypes.Burner},
		icqtypes.ModuleName:                nil,
		oraclemoduletypes.ModuleName:       {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		disputemoduletypes.ModuleName:      {authtypes.Minter, authtypes.Burner, authtypes.Staking},
		bridgemoduletypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		reportermoduletypes.ModuleName:     nil,
		reportermoduletypes.TipsEscrowPool: nil,
		// this line is used by starport scaffolding # stargate/app/maccPerms
	}
)

var (
	_ runtime.AppI            = (*App)(nil)
	_ servertypes.Application = (*App)(nil)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	version.AppName = "layer"
	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}

// App extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type App struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	AuthzKeeper           authzkeeper.Keeper
	BankKeeper            bankkeeper.Keeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICQKeeper             icqkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	GroupKeeper           groupkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper  capabilitykeeper.ScopedKeeper
	ScopedICQKeeper      capabilitykeeper.ScopedKeeper

	OracleKeeper oraclemodulekeeper.Keeper

	RegistryKeeper registrymodulekeeper.Keeper

	DisputeKeeper disputemodulekeeper.Keeper

	BridgeKeeper    bridgemodulekeeper.Keeper
	ReporterKeeper  reportermodulekeeper.Keeper
	GlobalFeeKeeper globalfeekeeper.Keeper
	// this line is used by starport scaffolding # stargate/app/keeperDeclaration

	// mm is the module manager
	mm                 *module.Manager
	BasicModuleManager module.BasicManager

	// sm is the simulation manager
	sm           *module.SimulationManager
	configurator module.Configurator
}

// New returns a reference to an initialized blockchain app
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
			},
			ValidatorAddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
			},
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)
	legacyAmino := codec.NewLegacyAmino()
	std.RegisterInterfaces(appCodec.InterfaceRegistry())
	std.RegisterLegacyAminoCodec(legacyAmino)

	bApp := baseapp.NewBaseApp(
		Name,
		logger,
		db,
		txConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey, authz.ModuleName, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey,
		feegrant.StoreKey, evidencetypes.StoreKey, ibctransfertypes.StoreKey, icahosttypes.StoreKey,
		capabilitytypes.StoreKey, group.StoreKey, icacontrollertypes.StoreKey, consensusparamtypes.StoreKey,
		oraclemoduletypes.StoreKey,
		registrymoduletypes.StoreKey,
		disputemoduletypes.StoreKey,
		bridgemoduletypes.StoreKey,
		reportermoduletypes.StoreKey,
		globalfeetypes.StoreKey,
		icqtypes.StoreKey,
		// this line is used by starport scaffolding # stargate/app/storeKey
	)

	memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))

	app := &App{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          txConfig,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		memKeys:           memKeys,
	}

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)

	// grant capabilities for the ibc and ibc-transfer modules
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedICAControllerKeeper := app.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	scopedICQKeeper := app.CapabilityKeeper.ScopeToModule(icqtypes.ModuleName)
	// this line is used by starport scaffolding # stargate/app/scopedKeeper

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[authz.ModuleName]),
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		app.BlockedModuleAccountAddrs(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)
	// https://docs.cosmos.network/main/build/migrations/upgrading
	// When using (legacy) application wiring, the following must be added to app.go after setting the app's bank keeper
	enabledSignModes := append(authtx.DefaultSignModes, sigtypes.SignMode_SIGN_MODE_TEXTUAL)
	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           enabledSignModes,
		TextualCoinMetadataQueryFn: txmodule.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}
	txConfig, err = authtx.NewTxConfigWithOptions(
		appCodec,
		txConfigOpts,
	)
	if err != nil {
		panic(fmt.Errorf("failed to create new TxConfig with options: %w", err))
	}
	app.txConfig = txConfig
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[feegrant.StoreKey]),
		app.AccountKeeper,
	)

	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.GlobalFeeKeeper = globalfeekeeper.NewKeeper(
		appCodec,
		app.keys[globalfeetypes.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	groupConfig := group.DefaultConfig()
	/*
		Example of setting group params:
		groupConfig.MaxMetadataLen = 1000
	*/
	app.GroupKeeper = groupkeeper.NewKeeper(
		keys[group.StoreKey],
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
		groupConfig,
	)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// ... other modules keepers

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibcexported.StoreKey],
		nil,
		app.StakingKeeper,
		app.UpgradeKeeper,
		scopedIBCKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		nil,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	transferIBCModule := transfer.NewIBCModule(app.TransferKeeper)

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey],
		nil,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		scopedICAHostKeeper,
		app.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	icaControllerKeeper := icacontrollerkeeper.NewKeeper(
		appCodec, keys[icacontrollertypes.StoreKey],
		nil,
		app.IBCKeeper.ChannelKeeper, // may be replaced with middleware such as ics29 fee
		app.IBCKeeper.ChannelKeeper, app.IBCKeeper.PortKeeper,
		scopedICAControllerKeeper, app.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)
	// ICQ Keeper
	app.ICQKeeper = icqkeeper.NewKeeper(
		appCodec,
		app.keys[icqtypes.StoreKey],
		app.IBCKeeper.ChannelKeeper, // may be replaced with middleware
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		scopedICQKeeper,
		bApp.GRPCQueryRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	icqModule := icq.NewIBCModule(app.ICQKeeper)
	// Create evidence Keeper for to register the IBC light client misbehavior evidence route
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(
		// register the governance hooks
		),
	)

	app.RegistryKeeper = registrymodulekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[registrymoduletypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ReporterKeeper = reportermodulekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[reportermoduletypes.StoreKey]),
		logger,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.AccountKeeper,
		app.StakingKeeper,
		app.BankKeeper,
		app.RegistryKeeper,
	)

	app.OracleKeeper = oraclemodulekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[oraclemoduletypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.RegistryKeeper,
		app.ReporterKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.DisputeKeeper = disputemodulekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[disputemoduletypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.OracleKeeper,
		app.ReporterKeeper,
	)

	app.BridgeKeeper = bridgemodulekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[bridgemoduletypes.StoreKey]),
		app.StakingKeeper,
		app.OracleKeeper,
		app.BankKeeper,
		app.ReporterKeeper,
		app.DisputeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.OracleKeeper.SetBridgeKeeper(app.BridgeKeeper)
	app.ReporterKeeper.SetOracleKeeper(app.OracleKeeper)
	// this line is used by starport scaffolding # stargate/app/keeperDefinition
	appFlags := appflags.GetFlagValuesFromOptions(appOpts)
	// Panic if this is not a full node and gRPC is disabled.
	if err := appFlags.Validate(); err != nil {
		panic(err)
	}

	/**** IBC Routing ****/

	// Sealing prevents other modules from creating scoped sub-keepers
	app.CapabilityKeeper.Seal()

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := ibcporttypes.NewRouter()
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule)
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferIBCModule)
	ibcRouter.AddRoute(icqtypes.ModuleName, icqModule)
	app.IBCKeeper.SetRouter(ibcRouter)

	/**** Module Hooks ****/

	// register hooks after all modules have been initialized

	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			// insert staking hooks receivers here
			app.DistrKeeper.Hooks(),
			app.SlashingKeeper.Hooks(),
			app.ReporterKeeper.Hooks(),
		),
	)

	voteExtHandler := NewVoteExtHandler(app.Logger(), app.AppCodec(), app.OracleKeeper, app.BridgeKeeper)
	app.BaseApp.SetExtendVoteHandler(voteExtHandler.ExtendVoteHandler)
	app.BaseApp.SetVerifyVoteExtensionHandler(voteExtHandler.VerifyVoteExtensionHandler)

	prepareProposalHandler := NewProposalHandler(app.Logger(), app.StakingKeeper, app.AppCodec(), app.OracleKeeper, app.BridgeKeeper, app.StakingKeeper)
	app.BaseApp.SetPrepareProposal(prepareProposalHandler.PrepareProposalHandler)
	app.BaseApp.SetProcessProposal(prepareProposalHandler.ProcessProposalHandler)
	app.BaseApp.SetPreBlocker(app.preBlocker(prepareProposalHandler))
	app.RegistryKeeper.SetHooks(
		registrymoduletypes.NewMultiRegistryHooks(
			app.OracleKeeper.Hooks(),
		),
	)

	/**** Module Options ****/
	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.

	app.mm = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper,
			app.StakingKeeper,
			app.BaseApp,
			txConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, nil),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bankModule{bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, nil)},
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		groupmodule.NewAppModule(appCodec, app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, nil),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, nil, app.interfaceRegistry),
		distrModule{distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, nil)},
		stakingModule{staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, nil)},
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),

		// Layer modules
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper),
		oraclemodule.NewAppModule(appCodec, app.OracleKeeper, app.AccountKeeper, app.BankKeeper),
		registrymodule.NewAppModule(appCodec, app.RegistryKeeper, app.AccountKeeper, app.BankKeeper),
		disputemodule.NewAppModule(appCodec, app.DisputeKeeper, app.AccountKeeper, app.BankKeeper),
		bridgemodule.NewAppModule(appCodec, app.BridgeKeeper, app.AccountKeeper, app.BankKeeper),
		reportermodule.NewAppModule(appCodec, app.ReporterKeeper, app.AccountKeeper, app.BankKeeper),
		// globalfee
		globalFeeModule{globalfee.NewAppModule(appCodec, app.GlobalFeeKeeper)},

		// IBC modules
		ibctm.AppModule{},
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ica.NewAppModule(&icaControllerKeeper, &app.ICAHostKeeper),
		icqcustomModule{icq.NewAppModule(app.ICQKeeper, nil)},
		// this line is used by starport scaffolding # stargate/app/appModule
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.mm,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(CustomMessageValidator),
			govtypes.ModuleName:     govModule{gov.NewAppModuleBasic([]govclient.ProposalHandler{})},
		})

	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(
		// upgrades should be run first
		upgradetypes.ModuleName,
		capabilitytypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		genutiltypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		group.ModuleName,
		paramstypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		oraclemoduletypes.ModuleName,
		registrymoduletypes.ModuleName,
		disputemoduletypes.ModuleName,
		bridgemoduletypes.ModuleName,
		reportermoduletypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/beginBlockers
	)

	app.mm.SetOrderEndBlockers(
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		ibctransfertypes.ModuleName,
		icqtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		group.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		oraclemoduletypes.ModuleName,
		registrymoduletypes.ModuleName,
		disputemoduletypes.ModuleName,
		bridgemoduletypes.ModuleName,
		reportermoduletypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/endBlockers
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	genesisModuleOrder := []string{
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		genutiltypes.ModuleName,
		ibctransfertypes.ModuleName,
		icqtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		evidencetypes.ModuleName,
		authz.ModuleName,
		feegrant.ModuleName,
		group.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		oraclemoduletypes.ModuleName,
		registrymoduletypes.ModuleName,
		disputemoduletypes.ModuleName,
		bridgemoduletypes.ModuleName,
		reportermoduletypes.ModuleName,
		globalfeetypes.ModuleName,
		// this line is used by starport scaffolding # stargate/app/initGenesis
	}
	app.mm.SetOrderInitGenesis(genesisModuleOrder...)
	app.mm.SetOrderExportGenesis(genesisModuleOrder...)

	// Uncomment if you want to set a custom migration order here.
	// app.mm.SetOrderMigrations(custom order)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err = app.mm.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	// RegisterUpgradeHandlers is used for registering any on-chain upgrades.
	// Make sure it's called after `app.ModuleManager` and `app.configurator` are set.
	app.setupUpgradeHandlers()
	app.setupUpgradeStoreLoaders()

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))
	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// create the simulation manager and define the order of the modules for deterministic simulations
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, nil),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)
	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountMemoryStores(memKeys)

	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setAnteHandler(txConfig)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper
	app.ScopedICQKeeper = scopedICQKeeper
	// this line is used by starport scaffolding # stargate/app/beforeInitReturn

	return app
}

func (app *App) setAnteHandler(txConfig client.TxConfig) {
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			ante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				SignModeHandler: txConfig.SignModeHandler(),
				FeegrantKeeper:  app.FeeGrantKeeper,
				SigGasConsumer:  ante.DefaultSigVerificationGasConsumer,
			},
			app.ReporterKeeper,
			app.StakingKeeper,
			app.GlobalFeeKeeper,
		},
	)
	if err != nil {
		panic(err)
	}

	// Set the AnteHandler for the app
	app.SetAnteHandler(anteHandler)
}

func (app *App) preBlocker(ph *ProposalHandler) func(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return func(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
		res, err := app.ModuleManager().PreBlock(ctx)
		if err != nil {
			return nil, err
		}
		if ctx.BlockHeight() == 1062219+1 {
			bondedPoolBal := app.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(stakingtypes.BondedPoolName), "loya")
			vals, err := app.StakingKeeper.GetValidators(ctx, 7) // only 6 validators exist
			if err != nil {
				return nil, err
			}
			bondedtotal := math.ZeroInt()
			notbondedtotal := math.ZeroInt()
			for _, v := range vals {
				if v.IsBonded() {
					bondedtotal = bondedtotal.Add(v.Tokens)
				} else {
					notbondedtotal = notbondedtotal.Add(v.Tokens)
				}
			}
			// transfer amount should be 1950000000
			// "spendable balance 97171770000loya is smaller than 99121770000loya
			transferAmt := bondedPoolBal.Amount.Sub(bondedtotal)
			if transferAmt.Equal(math.NewInt(1950000000)) { // manually checked amount
				err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, stakingtypes.BondedPoolName, stakingtypes.NotBondedPoolName, sdk.NewCoins(sdk.NewCoin("loya", transferAmt)))
				if err != nil {
					return nil, err
				}
			}
		}

		changed := res.ConsensusParamsChanged

		res, err = ph.PreBlocker(ctx, req)
		if changed != res.ConsensusParamsChanged {
			res.ConsensusParamsChanged = true
		}

		return res, err
	}
}

// Name returns the name of the App
func (app *App) Name() string { return app.BaseApp.Name() }

// GetBaseApp returns the base app of the application
func (app *App) GetBaseApp() *baseapp.BaseApp { return app.BaseApp }

// BeginBlocker application updates every begin block
func (app *App) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.mm.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.mm.EndBlock(ctx)
}

// Simple struct to hold upgrade plan in genesis
type GenesisUpgrade struct {
	Plans []upgradetypes.Plan `json:"plans,omitempty"`
}

// Helper function to create pre-upgrade version map
func getPreUpgradeVersionMap(currentVersions module.VersionMap) module.VersionMap {
	preUpgradeVersions := make(module.VersionMap)

	// Set modules that have migrations to their previous consensus version
	// You need to specify which modules should have their versions reduced
	for moduleName, currentVersion := range currentVersions {
		switch moduleName {
		case "dispute":
			// If dispute module upgraded from v3 to v4, set it to v3 for genesis
			preUpgradeVersions[moduleName] = currentVersion - 1
		case "oracle":
			// If oracle module upgraded from v2 to v3, set it to v2
			preUpgradeVersions[moduleName] = currentVersion - 1
		case "reporter":
			// If reporter module upgraded from v2 to v3, set it to v2
			preUpgradeVersions[moduleName] = currentVersion - 1
		// Add other modules that need migrations here
		default:
			// Modules without migrations keep current version
			preUpgradeVersions[moduleName] = currentVersion
		}
	}

	return preUpgradeVersions
}

// InitChainer application update at chain initialization
func (app *App) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	// Extract upgrade plan BEFORE initializing modules
	var upgradePlan *upgradetypes.Plan
	if upgradeData, exists := genesisState["upgrade"]; exists {
		var genUpgrade GenesisUpgrade
		if err := json.Unmarshal(upgradeData, &genUpgrade); err != nil {
			return nil, fmt.Errorf("failed to unmarshal upgrade genesis: %w", err)
		}

		if len(genUpgrade.Plans) > 0 {
			upgradePlan = &genUpgrade.Plans[0]
		}

		if upgradePlan != nil {
			// Validate the plan
			if upgradePlan.Name == "" {
				return nil, fmt.Errorf("upgrade plan name cannot be empty")
			}
			if upgradePlan.Height < 0 {
				return nil, fmt.Errorf("upgrade plan height cannot be negative")
			}

			app.Logger().Info("Found upgrade plan in genesis",
				"name", upgradePlan.Name,
				"height", upgradePlan.Height,
				"info", upgradePlan.Info)
		}
	}

	// Handle version map setup based on whether we have a genesis upgrade
	var initialVersionMap module.VersionMap
	if upgradePlan != nil && upgradePlan.Height <= ctx.BlockHeight() {
		// For genesis upgrades, set initial version map to simulate "before" state
		initialVersionMap = getPreUpgradeVersionMap(app.mm.GetVersionMap())
		app.Logger().Info("Setting pre-upgrade version map for genesis upgrade",
			"upgrade", upgradePlan.Name,
			"pre_versions", initialVersionMap)
	} else {
		// Normal initialization - use current module versions
		initialVersionMap = app.mm.GetVersionMap()
	}

	// Set the initial version map
	err := app.UpgradeKeeper.SetModuleVersionMap(ctx, initialVersionMap)
	if err != nil {
		return nil, err
	}

	// Initialize ALL modules first (including upgrade module)
	response, err := app.mm.InitGenesis(ctx, app.appCodec, genesisState)
	if err != nil {
		return nil, err
	}

	// AFTER all modules are initialized, handle the upgrade
	if upgradePlan != nil {
		// Check if this upgrade should execute at genesis initial height
		if upgradePlan.Height <= ctx.BlockHeight() {
			app.Logger().Info("Executing upgrade plan at genesis",
				"name", upgradePlan.Name,
				"planned_height", upgradePlan.Height,
				"current_height", ctx.BlockHeight())

			// Get the upgrade handler
			if hasHandler := app.UpgradeKeeper.HasHandler(upgradePlan.Name); !hasHandler {
				return nil, fmt.Errorf("no upgrade handler registered for '%s'", upgradePlan.Name)
			}

			// Execute the upgrade handler
			err := app.UpgradeKeeper.ApplyUpgrade(ctx, *upgradePlan)
			if err != nil {
				return nil, fmt.Errorf("failed to execute upgrade '%s': %w", upgradePlan.Name, err)
			}

			// Update the module version map after upgrade
			if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()); err != nil {
				return nil, fmt.Errorf("failed to set module version map after upgrade: %w", err)
			}

			app.Logger().Info("Successfully executed upgrade plan at genesis", "name", upgradePlan.Name)
		} else {
			// Schedule the upgrade for future execution
			app.Logger().Info("Scheduling upgrade plan for future execution",
				"name", upgradePlan.Name,
				"height", upgradePlan.Height)

			if err := app.UpgradeKeeper.ScheduleUpgrade(ctx, *upgradePlan); err != nil {
				return nil, fmt.Errorf("failed to schedule upgrade: %w", err)
			}
		}
	}

	return response, nil
}

// Configurator get app configurator
func (app *App) Configurator() module.Configurator {
	return app.configurator
}

// AutoCliOpts returns the autocli options for the app.
func (app *App) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.ModuleManager().Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.ModuleManager().Modules),
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// LoadHeight loads a particular height
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *App) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func (app *App) BlockedModuleAccountAddrs() map[string]bool {
	modAccAddrs := app.ModuleAccountAddrs()
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns an app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns an InterfaceRegistry
func (app *App) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns SimApp's TxConfig
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *App) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *App) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *App) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register app's OpenAPI routes.
	docs.RegisterOpenAPIService(Name, apiSvr.Router)
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService implements the Application.RegisterNodeService method.
func (app *App) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// SimulationManager returns the app SimulationManager
func (app *App) SimulationManager() *module.SimulationManager {
	return app.sm
}

// ModuleManager returns the app ModuleManager
func (app *App) ModuleManager() *module.Manager {
	return app.mm
}
