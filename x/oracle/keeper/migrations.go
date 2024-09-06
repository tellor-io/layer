package keeper

import (
	"github.com/tellor-io/layer/x/oracle/migrations"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a struct for handling in-place store migrations.
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
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return migrations.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc, m.keeper.Query)
}
