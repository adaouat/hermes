package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/adaouat/forge/updatecheck"
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
)

// jsonOutputCommands never get the update-check banner appended by PersistentPostRunE -
// it would corrupt their stdout, which Alfred/Raycast parse as JSON. doctor/install (M4)
// don't emit JSON and are deliberately left off this set so they keep the hint once they
// land (full finalization is roadmap M6's job).
var jsonOutputCommands = map[string]bool{
	"search": true,
	"all":    true,
	"open":   true,
}

func rootCmd(version string) *cobra.Command {
	rt := &runtime{}

	var launcherFlag, configFlag string
	var debug bool

	cmd := &cobra.Command{
		Use:           "hermes",
		Short:         "Locate JetBrains IDEs and recent projects for Alfred, Raycast, and other launchers",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(c *cobra.Command, _ []string) error {
			return rt.init(iofs.New(), env.New(), version, debug, launcherFlag, configFlag, c.ErrOrStderr())
		},
	}
	cmd.PersistentFlags().StringVar(&launcherFlag, "launcher", "", "Launcher output format: alfred, generic (default: auto-detect)")
	cmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to a jb_custom_config-shaped JSON file (overrides jb_custom_config)")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Verbose logging, mirrored to a temp file")

	cmd.PersistentPostRunE = func(c *cobra.Command, _ []string) error {
		if jsonOutputCommands[c.Name()] {
			return nil
		}
		updateHint(c, version)
		return nil
	}

	cmd.AddCommand(newSearchCmd(rt), newAllCmd(rt), newOpenCmd(rt), newConfigurationCmd(rt))

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
