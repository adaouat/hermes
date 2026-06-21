package main

import (
	"context"
	"os"

	"github.com/adaouat/forge/cli"
	forgeexit "github.com/adaouat/forge/exitcode"

	"github.com/adaouat/hermes/internal/ui"
)

var Version = "dev" // -ldflags "-X main.Version={{.Tag}}"

func main() {
	err := cli.Run(context.Background(), rootCmd(Version), Version, ui.Accent())
	os.Exit(forgeexit.Resolve(err))
}
