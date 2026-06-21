// Package iofs is the only door to the filesystem from internal/jetbrains and
// internal/launcher.
package iofs

import "io/fs"

// FS abstracts filesystem reads so callers are testable without touching the real disk.
type FS interface {
	Stat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	ReadFile(path string) ([]byte, error)
	Exists(path string) bool
	// Glob supports "*", "?", and a doublestar "**" segment that matches zero or more
	// path segments (needed for Fleet's backend/**/trusted-paths.xml lookup).
	Glob(pattern string) ([]string, error)
}

// New returns an FS backed by the real filesystem.
func New() FS {
	return osFS{}
}
