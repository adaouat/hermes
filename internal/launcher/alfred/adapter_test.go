package alfred

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestAdapter_Name(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	if got := a.Name(); got != "alfred" {
		t.Errorf("Name() = %q, want %q", got, "alfred")
	}
}

func TestAdapter_Detect(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want bool
	}{
		{name: "alfred_version set", vars: map[string]string{"alfred_version": "5.5"}, want: true},
		{name: "alfred_version unset", vars: map[string]string{}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := NewAdapter(envtest.New(nil))
			if got := a.Detect(envtest.New(tc.vars)); got != tc.want {
				t.Errorf("Detect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAdapter_Render_defaultTTL(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	var buf bytes.Buffer
	if err := a.Render(nil, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if got.Cache.Seconds != 86400 || !got.Cache.LooseReload {
		t.Errorf("Cache = %+v, want {86400 true}", got.Cache)
	}
}

func TestAdapter_Render_customTTL(t *testing.T) {
	a := NewAdapter(envtest.New(nil), WithTTLSeconds(60))
	var buf bytes.Buffer
	if err := a.Render(nil, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if got.Cache.Seconds != 60 {
		t.Errorf("Cache.Seconds = %d, want 60", got.Cache.Seconds)
	}
}

func TestAdapter_Render_appendsDebugItemsWithoutMutatingCaller(t *testing.T) {
	fixed := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	a := NewAdapter(
		envtest.New(map[string]string{"alfred_debug": "1"}),
		WithVersion("1.2.3"),
		WithClock(func() time.Time { return fixed }),
	)

	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}
	originalLen := len(items)

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}
	if len(items) != originalLen {
		t.Errorf("caller's items mutated: len = %d, want %d", len(items), originalLen)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("Items = %d, want 3 (1 project + 2 debug)", len(got.Items))
	}
	if got.Items[1].Title != "Debug: CLI version 1.2.3" {
		t.Errorf("Items[1].Title = %q, want %q", got.Items[1].Title, "Debug: CLI version 1.2.3")
	}
	if got.Items[2].Title != "Debug: Took 0ms" {
		t.Errorf("Items[2].Title = %q, want %q", got.Items[2].Title, "Debug: Took 0ms")
	}
}

func TestAdapter_Render_debugItemsWithEmptyVersion(t *testing.T) {
	fixed := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	a := NewAdapter(
		envtest.New(map[string]string{"alfred_debug": "1"}),
		WithClock(func() time.Time { return fixed }),
		// WithVersion not called — version is empty string by default
	)

	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("Items = %d, want 3 (1 project + 2 debug)", len(got.Items))
	}
	if got.Items[1].Title != "Debug: CLI version unknown" {
		t.Errorf("Items[1].Title = %q, want %q", got.Items[1].Title, "Debug: CLI version unknown")
	}
	if got.Items[1].Subtitle != "unknown" {
		t.Errorf("Items[1].Subtitle = %q, want %q", got.Items[1].Subtitle, "unknown")
	}
}

func TestAdapter_Render_noDebugItemsWithoutDebugEnv(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 1 {
		t.Errorf("Items = %d, want 1 (no debug items)", len(got.Items))
	}
}

func TestAdapter_Render_debugItemsIncludeLogFileWhenSet(t *testing.T) {
	fixed := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	a := NewAdapter(
		envtest.New(map[string]string{"alfred_debug": "1"}),
		WithClock(func() time.Time { return fixed }),
		WithLogFile("/tmp/hermes-debug-123.log"),
	)

	items := []domain.Item{{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", Match: "p p"}}

	var buf bytes.Buffer
	if err := a.Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got envelope
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal output: %v", err)
	}
	if len(got.Items) != 4 {
		t.Fatalf("Items = %d, want 4 (1 project + 3 debug)", len(got.Items))
	}
	if got.Items[3].Title != "Debug: Log /tmp/hermes-debug-123.log" {
		t.Errorf("Items[3].Title = %q, want %q", got.Items[3].Title, "Debug: Log /tmp/hermes-debug-123.log")
	}
}

func TestAdapter_InstallPropagatesAlfredNotInstalled(t *testing.T) {
	a := NewAdapter(envtest.New(nil))
	opts := launcher.InstallOpts{FS: iofstest.New(nil), Env: envtest.New(map[string]string{"HOME": "/home/x"})}
	if err := a.Install(context.Background(), opts); !errors.Is(err, ErrAlfredNotInstalled) {
		t.Errorf("Install() error = %v, want ErrAlfredNotInstalled", err)
	}
	if _, err := a.Verify(context.Background(), opts); !errors.Is(err, ErrAlfredNotInstalled) {
		t.Errorf("Verify() error = %v, want ErrAlfredNotInstalled", err)
	}
}
