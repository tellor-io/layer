package keeper

import (
	fork "github.com/tellor-io/layer/x/dispute/migrations/fork"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 migrates from version 2 to 3.
func (m Migrator) MigrateFork(ctx sdk.Context) error {
	return fork.MigrateStore(ctx,
		m.keeper.storeService,
		m.keeper.cdc,
	)
}
