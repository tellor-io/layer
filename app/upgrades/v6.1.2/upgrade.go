package v6_1_2

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	bridgekeeper "github.com/tellor-io/layer/x/bridge/keeper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"
	registrykeeper "github.com/tellor-io/layer/x/registry/keeper"
	registrytypes "github.com/tellor-io/layer/x/registry/types"

	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

/*
Upgrade to v6.1.2 includes:
  - Deprecation of TRBBridge query type in favor of TRBBridgeV2
  - Updates to oracle keeper to handle TRBBridgeV2 query type and prevent usage of TRBBridge
*/
// TODO: change these to real addresses
const (
	MainnetChainID       = "tellor1"
	MainnetTokenBridgeV2 = "0x0000000000000000000000000000000000000000"
	TestnetChainID       = "layertest-4"
	TestnetTokenBridgeV2 = "0x0000000000000000000000000000000000000000"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	oracleKeeper oraclekeeper.Keeper,
	bridgeKeeper bridgekeeper.Keeper,
	registryKeeper registrykeeper.Keeper,
) upgradetypes.UpgradeHandler {
	trbbridgeV2 := registrytypes.DataSpec{
		DocumentHash:      "",
		ResponseValueType: "address, string, uint256, uint256",
		AbiComponents: []*registrytypes.ABIComponent{
			{
				Name:            "toLayer",
				FieldType:       "bool",
				NestedComponent: []*registrytypes.ABIComponent{},
			},
			{
				Name:            "depositId",
				FieldType:       "uint256",
				NestedComponent: []*registrytypes.ABIComponent{},
			},
		},
		AggregationMethod: "weighted-mode",
		Registrar:         "genesis",
		ReportBlockWindow: 2000,
		QueryType:         "trbbridgev2",
	}

	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		err := registryKeeper.SetDataSpec(sdkCtx, "trbbridgev2", trbbridgeV2)
		if err != nil {
			return nil, err
		}

		tokenBridgeV2Address, err := resolveTokenBridgeV2Address(sdkCtx.ChainID())
		if err != nil {
			return nil, err
		}

		withdrawalId := uint64(1_000_000_000_000)
		recipient := common.HexToAddress(tokenBridgeV2Address).Bytes()
		sender := authtypes.NewModuleAddress(bridgetypes.ModuleName)
		amount := sdk.NewCoin("loya", math.NewInt(2_800_000_000_000)) // 2.8 million TRB - 1 TRB = 1_000_000 loya

		aggregate, queryData, err := bridgeKeeper.CreateWithdrawalAggregate(ctx, "TRBBridge", amount, sender, recipient, withdrawalId)
		if err != nil {
			return nil, err
		}
		err = oracleKeeper.SetAggregate(sdkCtx, aggregate, queryData, "TRBBridge-withdraw")
		if err != nil {
			return nil, err
		}
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

func resolveTokenBridgeV2Address(chainID string) (string, error) {
	switch chainID {
	case MainnetChainID:
		return MainnetTokenBridgeV2, nil
	case TestnetChainID:
		return TestnetTokenBridgeV2, nil
	default:
		addr := os.Getenv("TOKEN_BRIDGE_V2_ADDRESS")
		if addr == "" {
			return "", fmt.Errorf("TOKEN_BRIDGE_V2_ADDRESS env var not set for chain-id %s", chainID)
		}
		return addr, nil
	}
}
