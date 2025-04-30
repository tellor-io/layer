package keeper

import (
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/tellor-io/layer/x/reporter/migrations/fork"

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

// MigrateFork migrates from version to forked state
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
