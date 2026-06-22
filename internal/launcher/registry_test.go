package launcher

import (
	"context"
	"io"
	"testing"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/pkg/domain"
)

type fakeLauncher struct {
	name   string
	detect bool
}

func (f fakeLauncher) Name() string                               { return f.name }
func (f fakeLauncher) Detect(env.Env) bool                        { return f.detect }
func (f fakeLauncher) Render([]domain.Item, io.Writer) error      { return nil }
func (f fakeLauncher) Install(context.Context, InstallOpts) error { return nil }
func (f fakeLauncher) Verify(context.Context, InstallOpts) (Report, error) {
	return Report{}, nil
}

func TestRegistry_GetReturnsRegisteredLauncher(t *testing.T) {
	def := fakeLauncher{name: "generic"}
	alfred := fakeLauncher{name: "alfred"}
	r := NewRegistry(def, alfred)

	got, ok := r.Get("alfred")
	if !ok || got.Name() != "alfred" {
		t.Errorf("Get(%q) = (%v, %v), want (alfred, true)", "alfred", got, ok)
	}

	got, ok = r.Get("generic")
	if !ok || got.Name() != "generic" {
		t.Errorf("Get(%q) = (%v, %v), want (generic, true)", "generic", got, ok)
	}

	if _, ok := r.Get("missing"); ok {
		t.Errorf("Get(missing) ok = true, want false")
	}
}

func TestRegistry_Detect(t *testing.T) {
	tests := []struct {
		name     string
		others   []Launcher
		wantOK   bool
		wantName string
	}{
		{
			name:     "first matching launcher wins",
			others:   []Launcher{fakeLauncher{name: "alfred", detect: true}, fakeLauncher{name: "raycast", detect: true}},
			wantOK:   true,
			wantName: "alfred",
		},
		{
			name:   "no match returns false",
			others: []Launcher{fakeLauncher{name: "alfred", detect: false}},
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry(fakeLauncher{name: "generic"}, tc.others...)
			got, ok := r.Detect(envtest.New(nil))
			if ok != tc.wantOK {
				t.Fatalf("Detect() ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got.Name() != tc.wantName {
				t.Errorf("Detect() = %q, want %q", got.Name(), tc.wantName)
			}
		})
	}
}

func TestRegistry_Default(t *testing.T) {
	def := fakeLauncher{name: "generic"}
	r := NewRegistry(def, fakeLauncher{name: "alfred"})

	if got := r.Default(); got.Name() != "generic" {
		t.Errorf("Default() = %q, want %q", got.Name(), "generic")
	}
}
