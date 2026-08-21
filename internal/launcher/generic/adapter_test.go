package generic

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/launcher"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestAdapter_Name(t *testing.T) {
	if got := NewAdapter().Name(); got != "generic" {
		t.Errorf("Name() = %q, want %q", got, "generic")
	}
}

func TestAdapter_Detect_alwaysFalse(t *testing.T) {
	if got := NewAdapter().Detect(envtest.New(map[string]string{"alfred_version": "5"})); got {
		t.Errorf("Detect() = true, want false")
	}
}

func TestAdapter_Render(t *testing.T) {
	items := []domain.Item{
		{Name: "p", Path: "/x/p", IconPath: "/a.app", BinaryPath: "/a.app/Contents/MacOS/p", IsModernBinary: true, Match: "p p"},
	}

	var buf bytes.Buffer
	if err := NewAdapter().Render(items, &buf); err != nil {
		t.Fatalf("Render(): %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got) != 1 || got[0]["name"] != "p" || got[0]["binaryPath"] != "/a.app/Contents/MacOS/p" {
		t.Errorf("Render() = %v, want one item named %q", got, "p")
	}
}

func TestAdapter_InstallAndVerify(t *testing.T) {
	a := NewAdapter()
	if err := a.Install(context.Background(), launcher.InstallOpts{}); err != nil {
		t.Errorf("Install() error = %v, want nil", err)
	}
	report, err := a.Verify(context.Background(), launcher.InstallOpts{})
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
	if !report.Installed {
		t.Errorf("Verify().Installed = false, want true")
	}
}
