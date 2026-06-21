package iofs

import (
	"io/fs"
	"os"
	"path/filepath"
)

type osFS struct{}

func (osFS) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (osFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (osFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f osFS) Exists(path string) bool {
	_, err := f.Stat(path)
	return err == nil
}

func (f osFS) Glob(pattern string) ([]string, error) {
	if !hasDoubleStar(pattern) {
		return filepath.Glob(pattern)
	}

	baseDir := globBaseDir(pattern)
	if !f.Exists(baseDir) {
		return nil, nil
	}

	re := GlobRegexp(pattern)
	var matches []string
	err := filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && re.MatchString(path) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}
