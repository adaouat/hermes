package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/adaouat/forge/updatecheck"
)

func rootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hermes",
		Short:         "Locate JetBrains IDEs and recent projects for Alfred, Raycast, and other launchers",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentPostRunE = func(c *cobra.Command, _ []string) error {
		updateHint(c, version)
		return nil
	}
	// cmd.AddCommand(...)  ← search, all, open, install, configuration, doctor (roadmap M1-M4)
	return cmd
}

func updateHint(c *cobra.Command, version string) {
	if version == "dev" || os.Getenv("HERMES_CHECK_UPDATE") == "false" {
		return
	}
	cache, _ := os.UserCacheDir()
	updatecheck.Hinter{
		Repo:      "adaouat/hermes",
		Bin:       "hermes",
		Module:    "github.com/adaouat/hermes/cmd/hermes",
		Current:   version,
		CacheFile: filepath.Join(cache, "hermes", "update-check.json"),
	}.Print(c.Context(), c.ErrOrStderr())
}
