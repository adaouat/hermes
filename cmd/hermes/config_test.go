package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestLoadConfig_emptyPathFallsBackToEnv(t *testing.T) {
	e := envtest.New(map[string]string{"jb_custom_config": `{"phpStorm":{"preferencePrefix":"Custom"}}`})

	cfg, err := loadConfig(e, "")
	if err != nil {
		t.Fatalf("loadConfig(): %v", err)
	}
	if cfg[domain.PhpStorm].PreferencePrefix != "Custom" {
		t.Errorf("PreferencePrefix = %q, want %q", cfg[domain.PhpStorm].PreferencePrefix, "Custom")
	}
}

func TestLoadConfig_fileOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hermes-config.json")
	if err := os.WriteFile(path, []byte(`{"phpStorm":{"preferencePrefix":"FromFile"}}`), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	e := envtest.New(map[string]string{"jb_custom_config": `{"phpStorm":{"preferencePrefix":"FromEnv"}}`})

	cfg, err := loadConfig(e, path)
	if err != nil {
		t.Fatalf("loadConfig(): %v", err)
	}
	if cfg[domain.PhpStorm].PreferencePrefix != "FromFile" {
		t.Errorf("PreferencePrefix = %q, want %q", cfg[domain.PhpStorm].PreferencePrefix, "FromFile")
	}
}

func TestLoadConfig_missingFileReturnsError(t *testing.T) {
	if _, err := loadConfig(envtest.New(nil), "/nonexistent/hermes-config.json"); err == nil {
		t.Fatal("loadConfig(missing file): want error, got nil")
	}
}

func TestLoadConfig_malformedJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hermes-config.json")
	if err := os.WriteFile(path, []byte("not valid json"), 0o600); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}

	if _, err := loadConfig(envtest.New(nil), path); err == nil {
		t.Fatal("loadConfig(malformed JSON): want error, got nil")
	}
}
