package registry_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"
	registry "github.com/tellor-io/layer/x/registry/module"
)

func TestIsOnePerModuleType(t *testing.T) {
	appCodec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	am.IsOnePerModuleType()
}
func TestIsAppModule(t *testing.T) {
	appCodec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	am.IsAppModule()
}
func TestNewAppModuleBasic(t *testing.T) {
	appCodec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	appModule := registry.NewAppModuleBasic(appCodec)
	require.NotNil(t, appModule)
}
func TestName(t *testing.T) {
	appCodec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	require.Equal(t, "registry", am.Name())
}
func TestRegisterLegacyAminoCode(t *testing.T) {
	appCodec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	am := registry.NewAppModuleBasic(appCodec)
	cdc := codec.NewLegacyAmino()
	am.RegisterLegacyAminoCodec(cdc)
}

func TestDefaultGenesis(t *testing.T) {
}
func TestValidateGenesis(t *testing.T) {
}
func TestRegisterGRPCGatewayRoutes(t *testing.T) {
}
func TestNewAppModule(t *testing.T) {
}
func TestRegisterServioces(t *testing.T) {
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
