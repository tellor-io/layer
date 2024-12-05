package v2

import (
	"context"
	"fmt"

	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func MigrateStoreFromV1ToV2(ctx context.Context, keeper keeper.Keeper) error {

	err := keeper.SnapshotLimit.Set(ctx, types.SnapshotLimit{Limit: 1000})
	if err != nil {
		fmt.Println("error setting new snapshot limit: ", err)
		return err
	}

	return nil
}
