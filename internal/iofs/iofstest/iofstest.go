// Package iofstest provides an in-memory iofs.FS fake for tests.
package iofstest

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adaouat/hermes/internal/iofs"
)

// New returns an iofs.FS backed by the given map of absolute path -> file content,
// never the real disk. Directories are implied by file paths: registering
// "/a/b/c.txt" makes both "/a" and "/a/b" exist as directories.
func New(files map[string]string) iofs.FS {
	return fakeFS{files: files}
}

type fakeFS struct {
	files map[string]string
}

func (f fakeFS) isDir(path string) bool {
	prefix := filepath.Clean(path) + string(filepath.Separator)
	for p := range f.files {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

func (f fakeFS) Exists(path string) bool {
	clean := filepath.Clean(path)
	if _, ok := f.files[clean]; ok {
		return true
	}
	return f.isDir(clean)
}

func (f fakeFS) Stat(path string) (fs.FileInfo, error) {
	clean := filepath.Clean(path)
	if content, ok := f.files[clean]; ok {
		return fakeFileInfo{name: filepath.Base(clean), size: int64(len(content))}, nil
	}
	if f.isDir(clean) {
		return fakeFileInfo{name: filepath.Base(clean), isDir: true}, nil
	}
	return nil, &fs.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
}

func (f fakeFS) ReadFile(path string) ([]byte, error) {
	clean := filepath.Clean(path)
	content, ok := f.files[clean]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	return []byte(content), nil
}

func (f fakeFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	clean := filepath.Clean(dir)
	if !f.isDir(clean) {
		return nil, &fs.PathError{Op: "readdir", Path: dir, Err: fs.ErrNotExist}
	}

	prefix := clean + string(filepath.Separator)
	seen := map[string]fakeDirEntry{}
	for p, content := range f.files {
		if !strings.HasPrefix(p, prefix) {
			continue
		}
		rel := strings.TrimPrefix(p, prefix)
		name, isDir := rel, false
		if idx := strings.IndexRune(rel, filepath.Separator); idx != -1 {
			name, isDir = rel[:idx], true
		}
		if _, ok := seen[name]; !ok {
			seen[name] = fakeDirEntry{name: name, isDir: isDir, size: int64(len(content))}
		}
	}

	entries := make([]fs.DirEntry, 0, len(seen))
	for _, e := range seen {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (f fakeFS) Glob(pattern string) ([]string, error) {
	re := iofs.GlobRegexp(pattern)
	var matches []string
	for p := range f.files {
		if re.MatchString(p) {
			matches = append(matches, p)
		}
	}
	sort.Strings(matches)
	return matches, nil
}

type fakeFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (i fakeFileInfo) Name() string       { return i.name }
func (i fakeFileInfo) Size() int64        { return i.size }
func (i fakeFileInfo) Mode() fs.FileMode  { return 0o644 }
func (i fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeFileInfo) IsDir() bool        { return i.isDir }
func (i fakeFileInfo) Sys() any           { return nil }

type fakeDirEntry struct {
	name  string
	size  int64
	isDir bool
}

func (e fakeDirEntry) Name() string { return e.name }
func (e fakeDirEntry) IsDir() bool  { return e.isDir }
func (e fakeDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}
func (e fakeDirEntry) Info() (fs.FileInfo, error) {
	return fakeFileInfo(e), nil
}
