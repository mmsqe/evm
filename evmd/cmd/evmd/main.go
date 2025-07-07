package main

import (
	"fmt"
	"os"

	"github.com/cosmos/evm/evmd/cmd/evmd/cmd"
	evmdconfig "github.com/cosmos/evm/evmd/cmd/evmd/config"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/app"
)

func main() {
	setupSDKConfig()

	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "evmd", app.MustGetDefaultNodeHome()); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

func setupSDKConfig() {
	config := sdk.GetConfig()
	evmdconfig.SetBech32Prefixes(config)
	config.Seal()
}
