package jetbrains

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/internal/iofs"
	"github.com/adaouat/hermes/pkg/domain"
)

// Service is the single source of truth for turning a product's installed location and
// recent-projects list into launcher-neutral domain.Items - the orchestration that the
// legacy CLI duplicated between SearchCommand and SearchAllCommand ([smell] duplicated
// logic between SearchCommand and SearchAllCommand).
type Service struct {
	FS     iofs.FS
	Env    env.Env
	Config Config
}

// NewService returns a Service. Config is typically obtained once via Load(env) and
// reused across calls.
func NewService(fsys iofs.FS, e env.Env, cfg Config) Service {
	return Service{FS: fsys, Env: e, Config: cfg}
}

// Search returns product's recent projects as domain.Items, optionally filtered
// (case-insensitive) by project name or path basename. Any failure to locate the
// product's application, binary, or settings directory fails the whole call - the
// legacy CLI's per-project map() would similarly abort entirely on the first error
// rather than skip individual projects.
func (s Service) Search(product domain.Product, filter string) ([]domain.Item, error) {
	details, ok := s.Config[product]
	if !ok {
		return nil, fmt.Errorf("jetbrains: unknown product %q", product)
	}

	locator := NewLocator(s.FS, s.Env, product, details)
	appPath, err := locator.LocateApplication()
	if err != nil {
		return nil, err
	}
	binPath, err := locator.LocateBin()
	if err != nil {
		return nil, err
	}

	repo := NewRepository(s.FS, s.Env, product, details)
	projectPaths, err := repo.RecentProjects()
	if err != nil {
		return nil, err
	}

	names := NewProjectName(s.FS)
	filter = strings.ToLower(filter)
	isModernBinary := strings.Contains(binPath, "MacOS")

	items := make([]domain.Item, 0, len(projectPaths))
	for _, projectPath := range projectPaths {
		name, err := names.Resolve(projectPath)
		if err != nil {
			return nil, err
		}

		base := filepath.Base(projectPath)
		if filter != "" && !strings.Contains(strings.ToLower(name), filter) && !strings.Contains(strings.ToLower(base), filter) {
			continue
		}

		items = append(items, domain.Item{
			Name:           name,
			Path:           projectPath,
			IconPath:       appPath,
			BinaryPath:     binPath,
			IsModernBinary: isModernBinary,
			Match:          name + " " + base,
		})
	}
	return items, nil
}

// SearchAll aggregates Search across every supported product, silently skipping
// products that can't be located (e.g. not installed) rather than failing the whole
// call.
func (s Service) SearchAll(filter string) []domain.Item {
	var all []domain.Item
	for _, product := range domain.Products() {
		items, err := s.Search(product, filter)
		if err != nil {
			slog.Debug("skipping product in SearchAll", "product", product, "error", err)
			continue
		}
		all = append(all, items...)
	}
	return all
}
