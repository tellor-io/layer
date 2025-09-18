package keeper

import (
	v3 "github.com/tellor-io/layer/x/bridge/migrations/v3"
	v4 "github.com/tellor-io/layer/x/bridge/migrations/v4"
	v5 "github.com/tellor-io/layer/x/bridge/migrations/v5"
	"github.com/tellor-io/layer/x/bridge/types"

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
	// Migrate the ValidatorCheckpointParams store structure
	err := v4.MigrateStore(ctx,
		m.keeper.storeService,
		m.keeper.cdc,
	)
	if err != nil {
		return err
	}

	// Set the new bridge module parameters using Collections API
	// The Params structure existed before but was empty (no fields)
	defaultParams := types.DefaultParams()
	return m.keeper.Params.Set(ctx, defaultParams)
}

// Migrate4to5 migrates from version 4 to 5.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return v5.MigrateStore(ctx,
		&m.keeper,
		m.keeper.cdc,
	)
}
