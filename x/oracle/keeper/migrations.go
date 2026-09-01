package keeper

import (
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tellor-io/layer/x/oracle/migrations/fork"
	v4 "github.com/tellor-io/layer/x/oracle/migrations/v4"
	v5 "github.com/tellor-io/layer/x/oracle/migrations/v5"

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

// MigrateFork migrates from version 2 to 3.
func (m Migrator) MigrateFork(ctx sdk.Context) error {
	homeDir := viper.GetString("home")
	if homeDir == "" {
		panic("homeDir is empty, please use --home flag")
	}
	pathToFile := filepath.Join(
		homeDir,
		"config",
	)
	return fork.MigrateFork(ctx,
		m.keeper.storeService,
		m.keeper.cdc,
		pathToFile,
	)
}

// Migrate3to4 migrates from version 3 to 4.
// This adds the LivenessCycles parameter for liveness-weighted TBR distribution.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v4.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc)
}

// Migrate4to5 migrates from version 4 to 5.
// This deletes the unused MaxBatchSize store item after MsgBatchSubmitValue removal.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return v5.MigrateStore(ctx, m.keeper.storeService)
}
