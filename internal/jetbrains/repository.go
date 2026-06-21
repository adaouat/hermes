package jetbrains

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/pkg/domain"
)

var defaultSettingsPaths = []string{
	"~/Library/Application Support/Google",
	"~/Library/Application Support/JetBrains",
	"~/Library/Preferences",
}

// Repository finds an installed product's settings directory and recent-projects list.
type Repository struct {
	FS      iofs.FS
	Env     env.Env
	Product domain.Product
	Details ProductDetails
}

// NewRepository returns a Repository for product.
func NewRepository(fsys iofs.FS, e env.Env, product domain.Product, details ProductDetails) Repository {
	return Repository{FS: fsys, Env: e, Product: product, Details: details}
}

// LocateSettingsDirectory searches jb_settings (if set) or the default settings folders
// for the product's versioned preferences directory. Within each search path, candidate
// version directories are sorted descending and the first one with more than one child
// wins. A missing search path is skipped rather than treated as fatal ([bug]
// locateSettingsDirectory throws on first missing path).
func (r Repository) LocateSettingsDirectory() (string, error) {
	paths := settingsSearchPaths(r.Env)
	re := settingsRegexp(r.Product, r.Details.PreferencePrefix)

	for _, path := range paths {
		dir := parsePath(path, r.Env)
		if !r.FS.Exists(dir) {
			continue
		}
		entries, err := r.FS.ReadDir(dir)
		if err != nil {
			continue
		}

		type candidate struct {
			path    string
			version string
		}
		var candidates []candidate
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			m := re.FindStringSubmatch(entry.Name())
			if m == nil {
				continue
			}
			version := m[0]
			if len(m) > 1 {
				version = m[1]
			}
			candidates = append(candidates, candidate{path: filepath.Join(dir, entry.Name()), version: version})
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].version > candidates[j].version })

		for _, c := range candidates {
			children, err := r.FS.ReadDir(c.path)
			if err != nil {
				continue
			}
			if len(children) > 1 {
				return c.path, nil
			}
		}
	}

	return "", &NotFoundError{
		Product:       r.Product,
		What:          "settings directory",
		Names:         []string{r.Details.PreferencePrefix},
		SearchedPaths: paths,
	}
}

// RecentProjects returns the product's recent-project paths, following the legacy
// precedence: recentProjectDirectories.xml, then recentProjects.xml, then
// recentSolutions.xml (Rider), then - for Fleet only - any backend/**/trusted-paths.xml.
func (r Repository) RecentProjects() ([]string, error) {
	settingsDir, err := r.LocateSettingsDirectory()
	if err != nil {
		return nil, err
	}
	optionsDir := filepath.Join(settingsDir, "options")
	home := r.Env.Home()

	for _, f := range []struct {
		file    string
		extract func(content, home string) ([]string, error)
	}{
		{"recentProjectDirectories.xml", ExtractRecentProjectDirectories},
		{"recentProjects.xml", ExtractRecentProjects},
		{"recentSolutions.xml", ExtractRecentSolutions},
	} {
		path := filepath.Join(optionsDir, f.file)
		if !r.FS.Exists(path) {
			continue
		}
		content, err := r.FS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("jetbrains: reading %s: %w", path, err)
		}
		return f.extract(string(content), home)
	}

	if r.Product == domain.Fleet {
		return r.fleetTrustedPaths(settingsDir, home)
	}

	return nil, nil
}

func (r Repository) fleetTrustedPaths(settingsDir, home string) ([]string, error) {
	matches, err := r.FS.Glob(filepath.Join(settingsDir, "backend", "**", "trusted-paths.xml"))
	if err != nil {
		return nil, fmt.Errorf("jetbrains: globbing fleet trusted paths: %w", err)
	}

	var paths []string
	for _, match := range matches {
		content, err := r.FS.ReadFile(match)
		if err != nil {
			return nil, fmt.Errorf("jetbrains: reading %s: %w", match, err)
		}
		extracted, err := ExtractTrustedPaths(string(content), home)
		if err != nil {
			return nil, err
		}
		paths = append(paths, extracted...)
	}
	return paths, nil
}

func settingsSearchPaths(e env.Env) []string {
	v, ok := e.Lookup("jb_settings")
	if !ok || v == "" {
		return defaultSettingsPaths
	}
	return strings.Split(v, ":")
}

// settingsRegexp returns the per-product pattern matching a versioned settings
// directory basename, ported verbatim from projects.dart's switch on product:
//   - Fleet has no version suffix, so its preferences directory is matched by a bare
//     substring search with no anchors.
//   - Android Studio requires the year.quarter.fix form (e.g. "2024.3.2").
//   - Every other product requires year.quarter (e.g. "2024.1").
//
// Neither variant anchors the prefix on the left, matching the legacy behavior exactly.
func settingsRegexp(product domain.Product, prefix string) *regexp.Regexp {
	quoted := regexp.QuoteMeta(prefix)
	switch product {
	case domain.Fleet:
		return regexp.MustCompile(quoted)
	case domain.AndroidStudio:
		return regexp.MustCompile(quoted + `((\d|\d{4})\.\d(\.\d)$)`)
	default:
		return regexp.MustCompile(quoted + `((\d|\d{4})\.\d$)`)
	}
}
