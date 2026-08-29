package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/jetbrains"
)

// loadConfig returns the merged per-product configuration. An explicit --config path takes
// precedence over the jb_custom_config env var (jetbrains.Load); an empty path falls back
// to jetbrains.Load(e) so the frozen 2.x env-var contract keeps working untouched.
func loadConfig(e env.Env, path string) (jetbrains.Config, error) {
	if path == "" {
		return jetbrains.Load(e)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var custom map[string]any
	if err := json.Unmarshal(raw, &custom); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return jetbrains.Merge(jetbrains.Defaults(), custom)
}
