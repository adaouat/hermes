package jetbrains

import (
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/pkg/domain"
)

var defaultApplicationPaths = []string{
	"/Applications",
	"~/Applications",
	"~/Applications/JetBrains Toolbox",
}

// Locator finds an installed product's application bundle and binary on disk.
type Locator struct {
	FS      iofs.FS
	Env     env.Env
	Product domain.Product
	Details ProductDetails
}

// NewLocator returns a Locator for product, using details to know which application
// bundle names and binary names to search for.
func NewLocator(fsys iofs.FS, e env.Env, product domain.Product, details ProductDetails) Locator {
	return Locator{FS: fsys, Env: e, Product: product, Details: details}
}

// LocateApplication searches jb_application (if set) or the default Applications
// folders for an application bundle whose name (without the .app extension) matches one
// of Details.ApplicationNames. If multiple bundles match within the same directory, the
// first one (sorted by name) wins and the rest are logged as a warning - the legacy CLI
// threw in this situation ([bug] singleWhereOrNull throws on duplicates).
func (l Locator) LocateApplication() (string, error) {
	paths := applicationSearchPaths(l.Env)

	for _, path := range paths {
		dir := parsePath(path, l.Env)
		if !l.FS.Exists(dir) {
			continue
		}
		entries, err := l.FS.ReadDir(dir)
		if err != nil {
			continue
		}

		var matches []string
		for _, entry := range entries {
			base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			if slices.Contains(l.Details.ApplicationNames, base) {
				matches = append(matches, filepath.Join(dir, entry.Name()))
			}
		}
		if len(matches) > 0 {
			if len(matches) > 1 {
				slog.Warn("multiple application bundles matched, using first", "product", l.Product, "matches", matches)
			}
			return matches[0], nil
		}
	}

	return "", &NotFoundError{
		Product:       l.Product,
		What:          "application",
		Names:         l.Details.ApplicationNames,
		SearchedPaths: paths,
	}
}

// LocateBin searches jb_binaries (if set) or $PATH, plus the located application
// bundle's Contents/MacOS directory (the post-2023 DMG layout no longer ships bin
// scripts), for a binary whose name matches one of Details.Binaries. Same
// first-match-wins behavior as LocateApplication on duplicates within one directory.
func (l Locator) LocateBin() (string, error) {
	paths := binSearchPaths(l.Env)

	appPath, err := l.LocateApplication()
	if err != nil {
		return "", err
	}
	paths = append(paths, filepath.Join(appPath, "Contents", "MacOS"))

	for _, path := range paths {
		dir := parsePath(path, l.Env)
		if !l.FS.Exists(dir) {
			continue
		}
		entries, err := l.FS.ReadDir(dir)
		if err != nil {
			continue
		}

		var matches []string
		for _, entry := range entries {
			if slices.Contains(l.Details.Binaries, entry.Name()) {
				matches = append(matches, filepath.Join(dir, entry.Name()))
			}
		}
		if len(matches) > 0 {
			if len(matches) > 1 {
				slog.Warn("multiple binaries matched, using first", "product", l.Product, "matches", matches)
			}
			return matches[0], nil
		}
	}

	return "", &NotFoundError{
		Product:       l.Product,
		What:          "bin",
		Names:         l.Details.Binaries,
		SearchedPaths: paths,
	}
}

func applicationSearchPaths(e env.Env) []string {
	v, ok := e.Lookup("jb_application")
	if !ok || v == "" {
		return defaultApplicationPaths
	}
	return strings.Split(v, ":")
}

func binSearchPaths(e env.Env) []string {
	v, ok := e.Lookup("jb_binaries")
	if !ok || v == "" {
		return e.Path()
	}
	return strings.Split(v, ":")
}

// parsePath expands "~" to e.Home(). e.Home() returns "" when HOME is unset rather than
// panicking ([bug] force-unwrap of Platform.environment['HOME']!).
func parsePath(path string, e env.Env) string {
	if strings.Contains(path, "~") {
		return strings.ReplaceAll(path, "~", e.Home())
	}
	return path
}
