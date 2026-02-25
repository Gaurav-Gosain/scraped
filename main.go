package main

import (
	"context"
	"os"

	"charm.land/fang/v2"
	"github.com/Gaurav-Gosain/scraped/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		os.Exit(1)
	}
}
