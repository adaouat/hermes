package main

import (
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
)

func TestResolveLauncher(t *testing.T) {
	reg := newLauncherRegistry("1.0.0", envtest.New(nil), "")

	tests := []struct {
		name      string
		flagValue string
		envVars   map[string]string
		wantName  string
		wantErr   bool
	}{
		{name: "flag wins", flagValue: "alfred", wantName: "alfred"},
		{name: "env var used when no flag", envVars: map[string]string{"HERMES_LAUNCHER": "alfred"}, wantName: "alfred"},
		{name: "auto-detect when no flag or env", envVars: map[string]string{"alfred_version": "5.5"}, wantName: "alfred"},
		{name: "falls back to default", wantName: "generic"},
		{name: "unknown flag value errors", flagValue: "bogus", wantErr: true},
		{name: "unknown env value errors", envVars: map[string]string{"HERMES_LAUNCHER": "bogus"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := envtest.New(tc.envVars)
			got, err := resolveLauncher(tc.flagValue, e, reg)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveLauncher() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && got.Name() != tc.wantName {
				t.Errorf("resolveLauncher() = %q, want %q", got.Name(), tc.wantName)
			}
		})
	}
}
