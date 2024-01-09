package cmd

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func initSDKConfig() {
	// Set and seal config
	config := sdk.GetConfig()
	config.Seal()
}
