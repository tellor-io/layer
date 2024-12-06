package keeper

import (
	"context"

	v2 "github.com/tellor-io/layer/x/bridge/migrations/v2"
)

type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{
		keeper: keeper,
	}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx context.Context) error {
	return v2.MigrateStoreFromV1ToV2(ctx, m.keeper.storeService, m.keeper)
}
