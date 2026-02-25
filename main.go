package main

import (
	"context"
	"fmt"
	"os"

	"charm.land/fang/v2"
	"github.com/Gaurav-Gosain/scraped/cmd"
)

// Version information (set by goreleaser).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := cmd.NewRootCmd()
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(fmt.Sprintf("%s\nCommit: %s\nBuilt:  %s", version, commit, date)),
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		os.Exit(1)
	}
}
