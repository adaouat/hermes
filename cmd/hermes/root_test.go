package main

import "testing"

func TestRootCmd_registersAllCommands(t *testing.T) {
	cmd := rootCmd("1.0.0")

	for _, name := range []string{"search", "all", "open", "configuration", "install"} {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Errorf("rootCmd() missing subcommand %q: %v", name, err)
		}
	}
}
