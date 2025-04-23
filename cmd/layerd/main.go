package main

import (
	"fmt"
	"os"

	"github.com/tellor-io/layer/app"
	"github.com/tellor-io/layer/app/config"
	"github.com/tellor-io/layer/cmd/layerd/cmd"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
)

func main() {
	option := cmd.GetOptionWithCustomStartCmd()
	rootCmd := cmd.NewRootCmd(option)
	config.SetupConfig()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
