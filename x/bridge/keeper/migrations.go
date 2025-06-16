package keeper

import (
	v3 "github.com/tellor-io/layer/x/bridge/migrations/v3"
	v4 "github.com/tellor-io/layer/x/bridge/migrations/v4"

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
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v3.MigrateStore(ctx,
		m.keeper.storeService,
		m.keeper.cdc,
	)
}

// Migrate3to4 migrates from version 3 to 4.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v4.MigrateStore(ctx,
		m.keeper.storeService,
		m.keeper.cdc,
	)
}
