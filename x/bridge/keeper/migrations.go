package keeper

import (
	v3 "github.com/tellor-io/layer/x/bridge/migrations/v3"

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
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v3.MigrateStore(ctx,
		m.keeper.storeService,
		m.keeper.cdc,
	)
}
