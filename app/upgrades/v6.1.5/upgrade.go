package v6_1_5

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	bridgekeeper "github.com/tellor-io/layer/x/bridge/keeper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oraclekeeper "github.com/tellor-io/layer/x/oracle/keeper"

	"cosmossdk.io/collections"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.5 includes:
- Fix for mode calculation in x/oracle
*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	chainId string,
	bk bridgekeeper.Keeper,
	ok oraclekeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		chain := strings.EqualFold(chainId, "tellor-1")

		if chain {
			err := bk.DepositIdClaimedMap.Set(ctx, 146, bridgetypes.DepositClaimed{Claimed: false})
			if err != nil {
				return vm, fmt.Errorf("failed to set deposit claimed status: %w", err)
			}
			metaId := uint64(8183565)
			queryId := "57e934f27d9b60211c74816849e4b023246dd40a8ce6c1a7006f287ff10069ad"
			queryIdBytes, err := hex.DecodeString(queryId)
			if err != nil {
				return vm, fmt.Errorf("failed to decode query id: %w", err)
			}
			correctValueHex := "000000000000000000000000e18deb4baeddfbee123d10912a413460f0b35cf900000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000008ac7230489e800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d74656c6c6f7231686561767967727167326133683465646c6730617a6e6836687966366539743978666775666c00000000000000000000000000000000000000"
			aggregateTimestamp := uint64(1774909586032)
			aggregate, err := ok.GetAggregateByTimestamp(ctx, queryIdBytes, aggregateTimestamp)
			if err != nil {
				return vm, fmt.Errorf("failed to get aggregate by timestamp: %w", err)
			}
			aggregate.AggregateValue = correctValueHex
			err = ok.Aggregates.Set(ctx, collections.Join(queryIdBytes, aggregateTimestamp), aggregate)
			if err != nil {
				return vm, fmt.Errorf("failed to set aggregate: %w", err)
			}
			queryData, err := bk.EncodeDepositQueryData(146, "TRBBridge")
			if err != nil {
				return vm, fmt.Errorf("failed to encode deposit query data: %w", err)
			}
			err = ok.BridgeDepositQueue.Set(ctx, collections.Join(aggregateTimestamp, metaId), queryData)
			if err != nil {
				return vm, fmt.Errorf("failed to set bridge deposit queue: %w", err)
			}
		}
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
