package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/config"
	"github.com/tellor-io/layer/cmd/layerd/cmd"
)

func main() {
	config.SetupConfig()
	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
