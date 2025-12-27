package main

import (
	"os"

	"github.com/tabilabs/tabi-v3/app/params"
	"github.com/tabilabs/tabi-v3/cmd/tabid/cmd"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/tabilabs/tabi-v3/app"
)

func main() {
	params.SetAddressPrefixes()
	rootCmd, _ := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
