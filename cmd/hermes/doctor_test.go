package main

import (
	"bytes"
	"testing"
)

func TestRunDoctor_reportsFoundProductWithFullDetail(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runDoctor(&buf, rt, "phpStorm"); err != nil {
		t.Fatalf("runDoctor(): %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"PhpStorm",
		"found",
		"application: /Applications/PhpStorm.app",
		"binary: /Applications/PhpStorm.app/Contents/MacOS/phpstorm",
		"settings directory: /home/x/Library/Preferences/PhpStorm2024.1",
		"settings regex:",
		"recents file: /home/x/Library/Preferences/PhpStorm2024.1/options/recentProjectDirectories.xml",
	} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}

func TestRunDoctor_reportsNotFoundProduct(t *testing.T) {
	rt := newTestRuntime(nil)
	var buf bytes.Buffer

	if err := runDoctor(&buf, rt, "phpStorm"); err != nil {
		t.Fatalf("runDoctor(): %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("can't locate application")) {
		t.Errorf("output = %s, want a not-found explanation", out)
	}
}

func TestRunDoctor_settingsDirMissingStillReportsAppAndBin(t *testing.T) {
	files := map[string]string{
		"/Applications/PhpStorm.app/Contents/MacOS/phpstorm": "",
	}
	rt := newTestRuntime(files)
	var buf bytes.Buffer

	if err := runDoctor(&buf, rt, "phpStorm"); err != nil {
		t.Fatalf("runDoctor(): %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("application: /Applications/PhpStorm.app")) {
		t.Errorf("output missing application detail; got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("can't locate settings directory")) {
		t.Errorf("output missing settings-directory not-found explanation; got:\n%s", out)
	}
}

func TestRunDoctor_noRecentsFileYetReportsNone(t *testing.T) {
	files := map[string]string{
		"/Applications/PhpStorm.app/Contents/MacOS/phpstorm":     "",
		"/home/x/Library/Preferences/PhpStorm2024.1/other.txt":   "",
		"/home/x/Library/Preferences/PhpStorm2024.1/another.txt": "",
	}
	rt := newTestRuntime(files)
	var buf bytes.Buffer

	if err := runDoctor(&buf, rt, "phpStorm"); err != nil {
		t.Fatalf("runDoctor(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("recents file: none found")) {
		t.Errorf("output = %s, want a no-recents-yet report", buf.String())
	}
}

func TestRunDoctor_allProductsWhenNoFlag(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	var buf bytes.Buffer

	if err := runDoctor(&buf, rt, ""); err != nil {
		t.Fatalf("runDoctor(): %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("PhpStorm")) {
		t.Errorf("output missing PhpStorm; got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("GoLand")) {
		t.Errorf("output missing GoLand (should list every product); got:\n%s", out)
	}
}

func TestRunDoctor_unknownProductFlagErrors(t *testing.T) {
	rt := newTestRuntime(nil)
	var buf bytes.Buffer

	if err := runDoctor(&buf, rt, "notAProduct"); err == nil {
		t.Fatal("runDoctor(unknown product): want error, got nil")
	}
}

func TestNewDoctorCmd_executesEndToEnd(t *testing.T) {
	rt := newTestRuntime(phpStormFixture())
	cmd := newDoctorCmd(rt)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--product", "phpStorm"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("PhpStorm")) {
		t.Errorf("output = %s, want it to contain %q", buf.String(), "PhpStorm")
	}
}
