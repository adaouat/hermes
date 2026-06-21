package iofstest

import (
	"errors"
	"io/fs"
	"sort"
	"testing"
)

func TestNew_Exists(t *testing.T) {
	f := New(map[string]string{
		"/home/x/proj/.idea/name": "MyProject",
	})

	if !f.Exists("/home/x/proj/.idea/name") {
		t.Errorf("Exists(file) = false, want true")
	}
	if !f.Exists("/home/x/proj/.idea") {
		t.Errorf("Exists(implied dir) = false, want true")
	}
	if !f.Exists("/home/x/proj") {
		t.Errorf("Exists(implied grandparent dir) = false, want true")
	}
	if f.Exists("/home/x/missing") {
		t.Errorf("Exists(missing) = true, want false")
	}
}

func TestNew_ReadFile(t *testing.T) {
	f := New(map[string]string{
		"/home/x/proj/.idea/name": "MyProject",
	})

	got, err := f.ReadFile("/home/x/proj/.idea/name")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "MyProject" {
		t.Errorf("ReadFile = %q, want %q", got, "MyProject")
	}

	if _, err := f.ReadFile("/home/x/missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestNew_Stat(t *testing.T) {
	f := New(map[string]string{
		"/home/x/proj/.idea/name": "MyProject",
	})

	fileInfo, err := f.Stat("/home/x/proj/.idea/name")
	if err != nil {
		t.Fatalf("Stat(file): %v", err)
	}
	if fileInfo.IsDir() {
		t.Errorf("Stat(file).IsDir() = true, want false")
	}

	dirInfo, err := f.Stat("/home/x/proj/.idea")
	if err != nil {
		t.Fatalf("Stat(dir): %v", err)
	}
	if !dirInfo.IsDir() {
		t.Errorf("Stat(dir).IsDir() = false, want true")
	}

	if _, err := f.Stat("/home/x/missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestNew_ReadDir(t *testing.T) {
	f := New(map[string]string{
		"/apps/PhpStorm2024.1/options/a.xml":   "",
		"/apps/PhpStorm2024.1/options/b.xml":   "",
		"/apps/PhpStorm2024.1/other/c.xml":     "",
		"/apps/PhpStorm2023.3/options/old.xml": "",
	})

	entries, err := f.ReadDir("/apps")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
		if !e.IsDir() {
			t.Errorf("entry %q IsDir() = false, want true", e.Name())
		}
	}
	sort.Strings(names)
	want := []string{"PhpStorm2023.3", "PhpStorm2024.1"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Errorf("ReadDir names = %v, want %v", names, want)
	}

	if _, err := f.ReadDir("/missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadDir(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestNew_Glob(t *testing.T) {
	f := New(map[string]string{
		"/settings/backend/p1/trusted-paths.xml":        "",
		"/settings/backend/p2/nested/trusted-paths.xml": "",
		"/settings/backend/other.xml":                   "",
	})

	matches, err := f.Glob("/settings/backend/**/trusted-paths.xml")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	sort.Strings(matches)
	want := []string{
		"/settings/backend/p1/trusted-paths.xml",
		"/settings/backend/p2/nested/trusted-paths.xml",
	}
	if len(matches) != len(want) || matches[0] != want[0] || matches[1] != want[1] {
		t.Errorf("Glob = %v, want %v", matches, want)
	}
}
