package iofs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func TestNew_Exists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "hello")

	f := New()

	if !f.Exists(filepath.Join(dir, "a.txt")) {
		t.Errorf("Exists(a.txt) = false, want true")
	}
	if !f.Exists(dir) {
		t.Errorf("Exists(dir) = false, want true")
	}
	if f.Exists(filepath.Join(dir, "missing.txt")) {
		t.Errorf("Exists(missing.txt) = true, want false")
	}
}

func TestNew_ReadFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "hello")
	f := New()

	got, err := f.ReadFile(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("ReadFile = %q, want %q", got, "hello")
	}

	if _, err := f.ReadFile(filepath.Join(dir, "missing.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestNew_Stat(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "hello")
	f := New()

	info, err := f.Stat(filepath.Join(dir, "a.txt"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.IsDir() {
		t.Errorf("Stat(a.txt).IsDir() = true, want false")
	}

	dirInfo, err := f.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(dir): %v", err)
	}
	if !dirInfo.IsDir() {
		t.Errorf("Stat(dir).IsDir() = false, want true")
	}

	if _, err := f.Stat(filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestNew_ReadDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "")
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "")
	f := New()

	entries, err := f.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	want := []string{"a.txt", "sub"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Errorf("ReadDir names = %v, want %v", names, want)
	}
}

func TestNew_Glob(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "backend", "p1", "trusted-paths.xml"), "")
	writeFile(t, filepath.Join(dir, "backend", "p2", "nested", "trusted-paths.xml"), "")
	writeFile(t, filepath.Join(dir, "backend", "other.xml"), "")
	f := New()

	matches, err := f.Glob(filepath.Join(dir, "backend", "**", "trusted-paths.xml"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	sort.Strings(matches)
	want := []string{
		filepath.Join(dir, "backend", "p1", "trusted-paths.xml"),
		filepath.Join(dir, "backend", "p2", "nested", "trusted-paths.xml"),
	}
	sort.Strings(want)
	if len(matches) != len(want) || matches[0] != want[0] || matches[1] != want[1] {
		t.Errorf("Glob = %v, want %v", matches, want)
	}
}

func TestNew_Glob_noMatches(t *testing.T) {
	dir := t.TempDir()
	f := New()

	matches, err := f.Glob(filepath.Join(dir, "backend", "**", "trusted-paths.xml"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("Glob = %v, want empty", matches)
	}
}
