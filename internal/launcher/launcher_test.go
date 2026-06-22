package launcher

import "testing"

func TestInstallOpts_zeroValue(t *testing.T) {
	var opts InstallOpts
	if opts.DryRun || opts.Force {
		t.Errorf("zero InstallOpts has a true bool field: %+v", opts)
	}
	if opts.BinaryPath != "" || opts.Version != "" {
		t.Errorf("zero InstallOpts has a non-empty string field: %+v", opts)
	}
	if opts.FS != nil || opts.Env != nil || opts.Out != nil {
		t.Errorf("zero InstallOpts has a non-nil interface field: %+v", opts)
	}
}

func TestReport_fields(t *testing.T) {
	r := Report{Installed: true, Path: "/path", Drift: []string{"a", "b"}}
	if !r.Installed {
		t.Errorf("Installed = false, want true")
	}
	if r.Path != "/path" {
		t.Errorf("Path = %q, want %q", r.Path, "/path")
	}
	if len(r.Drift) != 2 {
		t.Errorf("Drift = %v, want 2 entries", r.Drift)
	}
}
