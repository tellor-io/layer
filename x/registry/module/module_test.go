package registry_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	registry "github.com/tellor-io/layer/x/registry/module"
	"github.com/tellor-io/layer/x/registry/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkTypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func TestIsOnePerModuleType(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	am.IsOnePerModuleType()
}

func TestIsAppModule(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	am.IsAppModule()
}

func TestNewAppModuleBasic(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	appModule := registry.NewAppModuleBasic(appCodec)
	require.NotNil(t, appModule)
}

func TestName(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	require.Equal(t, "registry", am.Name())
}

func TestRegisterLegacyAminoCode(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	cdc := codec.NewLegacyAmino()
	am.RegisterLegacyAminoCodec(cdc)
}

func TestDefaultGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:   types.DefaultParams(),
		Dataspec: types.GenesisDataSpec(),
	}
	k, _, _, ctx := keepertest.RegistryKeeper(t)
	registry.InitGenesis(ctx, k, genesisState)
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	initGenesis := am.DefaultGenesis(appCodec)
	require.NotNil(t, initGenesis)
}

func TestValidateGenesis(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	h := json.RawMessage(`{
        "params": {
			"max_report_buffer_window": "100000"
		}
	}`)

	err := am.ValidateGenesis(appCodec, nil, h)
	require.NoError(t, err)
}

func TestRegisterGRPCGatewayRoutes(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	router := runtime.NewServeMux()
	am.RegisterGRPCGatewayRoutes(client.Context{}, router)
	// Expect EventParams route registered
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/layer/registry/event_params", nil)
	fmt.Println(req)
	require.NoError(t, err)
	router.ServeHTTP(recorder, req)
}

func TestNewAppModule(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	k, _k2, _k3, _ := keepertest.RegistryKeeper(t)
	am := registry.NewAppModule(appCodec, k, _k2, _k3)
	require.NotNil(t, am)
}

func TestRegisterServices(t *testing.T) {
	// mockConfigurator := new(mocks.Configurator)
	// mockQueryServer := new(mocks.Server)
	// mockMsgServer := new(mocks.Server)

	// mockConfigurator.On("QueryServer").Return(mockQueryServer)
	// mockConfigurator.On("MsgServer").Return(mockMsgServer)
	// mockQueryServer.On("RegisterService", mock.Anything, mock.Anything).Return()
	// mockMsgServer.On("RegisterService", mock.Anything, mock.Anything).Return()

	// am := createAppModule(t)
	// am.RegisterServices(mockConfigurator)

	// require.Equal(t, true, mockConfigurator.AssertExpectations(t))
	// require.Equal(t, true, mockQueryServer.AssertExpectations(t))
	// require.Equal(t, true, mockMsgServer.AssertExpectations(t))
}

func TestRegisterInvariants(t *testing.T) {
}

func TestInitGenesis(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	k, _k2, _k3, ctx := keepertest.RegistryKeeper(t)
	am := registry.NewAppModule(appCodec, k, _k2, _k3)
	genesisState := types.DefaultGenesis()
	json, err := json.Marshal(genesisState)
	require.NoError(t, err)
	am.InitGenesis(ctx, appCodec, json)
}

func TestExportGenesis(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	k, _k2, _k3, ctx := keepertest.RegistryKeeper(t)
	am := registry.NewAppModule(appCodec, k, _k2, _k3)
	defaultGen := types.DefaultGenesis()
	fmt.Println("genesisState: ", defaultGen)
	j, err := json.Marshal(defaultGen)
	require.NoError(t, err)

	am.InitGenesis(ctx, appCodec, j)
	gen := am.ExportGenesis(ctx, appCodec)
	fmt.Println("exported genesis: ", gen)
	fmt.Println("expected: ", j)
	genJson, err := json.Marshal(gen)
	fmt.Println("genJson: ", genJson)
	unmarshalled := json.Unmarshal(genJson, &types.GenesisState{})
	fmt.Println("unmarshalled: ", unmarshalled)
	require.NoError(t, err)
	require.Equal(t, genJson, j)
}

func TestConsensusVersion(t *testing.T) {
	appCodec := codec.NewProtoCodec(sdkTypes.NewInterfaceRegistry())
	k, _k2, _k3, _ := keepertest.RegistryKeeper(t)
	am := registry.NewAppModule(appCodec, k, _k2, _k3)
	val := am.ConsensusVersion()
	require.Equal(t, int(val), 1)
}
