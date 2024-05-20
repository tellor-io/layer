package registry_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	registry "github.com/tellor-io/layer/x/registry/module"
	"github.com/tellor-io/layer/x/registry/types"
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
	k, ctx := keepertest.RegistryKeeper(t)
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
		"params": {}
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
	//require.Contains(t, recorder.Body.String(), "no RPC client is defined in offline mode")
}
func TestNewAppModule(t *testing.T) {
}

func TestRegisterServices(t *testing.T) {

}

func TestRegisterInvariants(t *testing.T) {
}

func TestInitGenesis(t *testing.T) {
}
func TestExportGenesis(t *testing.T) {
}
func TestConsensusVersion(t *testing.T) {
}
func TestInit(t *testing.T) {
}
func TestProvideModule(t *testing.T) {
}
