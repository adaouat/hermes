package main

import (
	"github.com/spf13/cobra"

	"github.com/adaouat/forge/updatecheck"
	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
)

// jsonOutputCommands never get the update-check banner appended by PersistentPostRunE -
// it avoids a noisy version-upgrade banner appearing on stderr during Alfred/Raycast-driven
// invocations (stderr can surface in debugger panels, which would be confusing). doctor/install
// (M4) don't emit JSON and are deliberately left off this set so they keep the hint once they
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
	cmd.PersistentFlags().StringVar(&launcherFlag, "launcher", "", "Launcher output format: alfred, raycast, generic (default: auto-detect)")
	cmd.PersistentFlags().StringVar(&configFlag, "config", "", "Path to a jb_custom_config-shaped JSON file (overrides jb_custom_config)")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Verbose logging, mirrored to a temp file (or set HERMES_DEBUG)")

	// Hinter.PostRun() already gates on version=="dev" and the OptOutEnv var (forge
	// updatecheck.Hinter's own doc comment); jsonOutputCommands is the one gate that's
	// hermes-specific (Alfred/Raycast/generic all read stdout as a JSON envelope, so the
	// hint - like any stderr banner - must never fire there even though it wouldn't
	// corrupt stdout directly, since it can still surface in a debugger panel).
	hinter := updatecheck.Hinter{
		Repo:      "adaouat/hermes",
		Bin:       "hermes",
		Module:    "github.com/adaouat/hermes/cmd/hermes",
		Current:   version,
		CacheFile: updatecheck.CacheFile("hermes"),
		OptOutEnv: "HERMES_CHECK_UPDATE",
	}
	cmd.PersistentPostRunE = func(c *cobra.Command, args []string) error {
		if jsonOutputCommands[c.Name()] {
			return nil
		}
		return hinter.PostRun()(c, args)
	}

	cmd.AddCommand(newSearchCmd(rt), newAllCmd(rt), newOpenCmd(rt), newConfigurationCmd(rt), newInstallCmd(rt, version), newDoctorCmd(rt))

	return cmd
}
