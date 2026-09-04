// Package test proves the Go binary is a drop-in for the 2.6.2 Dart CLI (roadmap M7): it
// exercises the real internal/iofs, internal/env, internal/jetbrains, and
// internal/launcher/alfred implementations - no fakes - against a committed fake macOS home
// (test/fixtures/home/), and asserts byte-identity against JSON captured from a real,
// freshly-compiled alfred_jetbrains_cli 2.6.2 binary against that exact same fixture
// (test/fixtures/golden/, docs/adr/0008-golden-parity-test-contract.md).
package test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/internal/launcher/alfred"
	"github.com/adaouat/hermes/pkg/domain"
)

// fixtureHome sets HOME, jb_application, jb_settings, and jb_binaries (via t.Setenv, so the
// real process env is restored on test exit per testing.md's determinism rule) to isolate
// the real env.New()/iofs.New() implementations to test/fixtures/home/ - never the real
// $HOME or /Applications. jb_binaries points at a path that doesn't exist so LocateBin can
// only ever resolve a fixture binary under the located .app's Contents/MacOS, regardless of
// what's really on the host machine's $PATH (the exact hazard M4's doctor smoke test hit).
func fixtureHome(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("fixtures/home")
	if err != nil {
		t.Fatalf("resolving fixture home: %v", err)
	}
	t.Setenv("HOME", abs)
	t.Setenv("jb_application", filepath.Join(abs, "Applications"))
	t.Setenv("jb_settings", filepath.Join(abs, "Library", "Application Support", "JetBrains"))
	t.Setenv("jb_binaries", filepath.Join(abs, ".nonexistent-binaries"))
	return abs
}

// wantGolden reads testdata name (fixtures/golden/<name>.json), substituting the {{HOME}}
// placeholder (how the fixture's own machine-specific absolute path was templated out when
// captured) with home.
func wantGolden(t *testing.T, name, home string) string {
	t.Helper()
	path := filepath.Join("fixtures", "golden", name+".json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden fixture %s: %v", path, err)
	}
	want := bytes.ReplaceAll(content, []byte("{{HOME}}"), []byte(home))
	return string(want)
}

func TestGoldenParity_Search(t *testing.T) {
	tests := []struct {
		name    string
		product domain.Product
		golden  string
	}{
		{"phpStorm - standard case", domain.PhpStorm, "search_phpstorm"},
		{"androidStudio - year.quarter.fix settings-dir regex", domain.AndroidStudio, "search_androidstudio"},
		{"fleet - unanchored settings-dir substring, trusted-paths.xml", domain.Fleet, "search_fleet"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := fixtureHome(t)
			e := env.New()
			svc := jetbrains.NewService(iofs.New(), e, jetbrains.Defaults())

			items, err := svc.Search(tc.product, "")
			if err != nil {
				t.Fatalf("Search(%s): %v", tc.product, err)
			}

			var buf bytes.Buffer
			if err := alfred.NewAdapter(e).Render(items, &buf); err != nil {
				t.Fatalf("Render(): %v", err)
			}

			want := wantGolden(t, tc.golden, home)
			if buf.String() != want {
				t.Errorf("search %s output mismatch:\ngot:  %s\nwant: %s", tc.product, buf.String(), want)
			}
		})
	}
}

func TestGoldenParity_SearchAll(t *testing.T) {
	home := fixtureHome(t)
	e := env.New()
	svc := jetbrains.NewService(iofs.New(), e, jetbrains.Defaults())

	items := svc.SearchAll("")

	var buf bytes.Buffer
	if err := alfred.NewAdapter(e).Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	want := wantGolden(t, "all", home)
	if buf.String() != want {
		t.Errorf("all output mismatch:\ngot:  %s\nwant: %s", buf.String(), want)
	}
}

// TestGoldenParity_Open proves the deliberate divergence ADR-0004 documents: the legacy
// CLI's "open" skipped the {"cache":...,"items":[...]} envelope ([bug] renderItem vs
// renderItems shape mismatch); the Go rewrite's single Render path always uses it. The
// golden fixture here is the *enveloped* shape Render actually produces, not a byte-capture
// of the legacy CLI's unenveloped "open" output.
func TestGoldenParity_Open(t *testing.T) {
	home := fixtureHome(t)
	e := env.New()
	svc := jetbrains.NewService(iofs.New(), e, jetbrains.Defaults())

	item, err := svc.Open(domain.PhpStorm, filepath.Join(home, "projects", "AuroraProject"))
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}

	var buf bytes.Buffer
	if err := alfred.NewAdapter(e).Render([]domain.Item{item}, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	want := wantGolden(t, "open_phpstorm_enveloped", home)
	if buf.String() != want {
		t.Errorf("open output mismatch:\ngot:  %s\nwant: %s", buf.String(), want)
	}
}
