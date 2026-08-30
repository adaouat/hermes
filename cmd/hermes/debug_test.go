package main

import (
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
)

func TestResolveDebug(t *testing.T) {
	tests := []struct {
		name      string
		flagValue bool
		envVars   map[string]string
		want      bool
	}{
		{name: "flag wins when true", flagValue: true, want: true},
		{name: "env var used when no flag", envVars: map[string]string{"HERMES_DEBUG": "1"}, want: true},
		{name: "env var presence is enough, value ignored", envVars: map[string]string{"HERMES_DEBUG": ""}, want: true},
		{name: "neither set", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := envtest.New(tc.envVars)
			if got := resolveDebug(tc.flagValue, e); got != tc.want {
				t.Errorf("resolveDebug(%v, %v) = %v, want %v", tc.flagValue, tc.envVars, got, tc.want)
			}
		})
	}
}
