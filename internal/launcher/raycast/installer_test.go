package raycast

import (
	"bytes"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/launcher"
)

func TestInstall_reportsPathReachability(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		pathVar    string
		wantSubstr string
	}{
		{
			name:       "binary dir on PATH",
			binaryPath: "/opt/homebrew/bin/hermes",
			pathVar:    "/usr/bin:/opt/homebrew/bin",
			wantSubstr: "reachable on $PATH",
		},
		{
			name:       "binary dir not on PATH",
			binaryPath: "/opt/homebrew/bin/hermes",
			pathVar:    "/usr/bin",
			wantSubstr: "not on $PATH",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			opts := launcher.InstallOpts{
				BinaryPath: tc.binaryPath,
				Env:        envtest.New(map[string]string{"PATH": tc.pathVar}),
				Out:        &out,
			}
			if err := install(opts); err != nil {
				t.Fatalf("install(): %v", err)
			}
			if !bytes.Contains(out.Bytes(), []byte(tc.wantSubstr)) {
				t.Errorf("install() output = %q, want substring %q", out.String(), tc.wantSubstr)
			}
		})
	}
}

func TestInstall_dryRunWritesNothingDestructive(t *testing.T) {
	var out bytes.Buffer
	opts := launcher.InstallOpts{
		DryRun:     true,
		BinaryPath: "/opt/homebrew/bin/hermes",
		Env:        envtest.New(map[string]string{"PATH": "/opt/homebrew/bin"}),
		Out:        &out,
	}
	if err := install(opts); err != nil {
		t.Fatalf("install(): %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("would verify")) {
		t.Errorf("install(DryRun) output = %q, want substring %q", out.String(), "would verify")
	}
}

func TestVerify_reachableOnPath(t *testing.T) {
	opts := launcher.InstallOpts{
		BinaryPath: "/opt/homebrew/bin/hermes",
		Env:        envtest.New(map[string]string{"PATH": "/usr/bin:/opt/homebrew/bin"}),
	}
	report, err := verify(opts)
	if err != nil {
		t.Fatalf("verify(): %v", err)
	}
	if !report.Installed || report.Path != "/opt/homebrew/bin" || len(report.Drift) != 0 {
		t.Errorf("verify() = %+v, want Installed=true, Path=/opt/homebrew/bin, no drift", report)
	}
}

func TestVerify_notOnPathReportsDrift(t *testing.T) {
	opts := launcher.InstallOpts{
		BinaryPath: "/opt/homebrew/bin/hermes",
		Env:        envtest.New(map[string]string{"PATH": "/usr/bin"}),
	}
	report, err := verify(opts)
	if err != nil {
		t.Fatalf("verify(): %v", err)
	}
	if report.Installed || len(report.Drift) == 0 {
		t.Errorf("verify() = %+v, want Installed=false with drift", report)
	}
}
