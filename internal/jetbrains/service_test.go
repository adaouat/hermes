package jetbrains

import (
	"errors"
	"slices"
	"sort"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/pkg/domain"
)

func phpStormFixture() map[string]string {
	return map[string]string{
		"/Applications/PhpStorm.app/Contents/MacOS/phpstorm":   "",
		"/home/x/Library/Preferences/PhpStorm2024.1/other.txt": "",
		"/home/x/Library/Preferences/PhpStorm2024.1/options/recentProjectDirectories.xml": `<application>
  <component name="RecentDirectoryProjectsManager">
    <option name="recentPaths">
      <list>
        <option value="$USER_HOME$/projects/aurora" />
        <option value="$USER_HOME$/projects/beacon" />
      </list>
    </option>
  </component>
</application>`,
		"/home/x/projects/aurora/.idea/name": "AuroraProject",
		"/home/x/projects/beacon/README.md":  "",
	}
}

func TestService_Search_buildsItems(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(phpStormFixture()), e, Defaults())

	items, err := svc.Search(domain.PhpStorm, "")
	if err != nil {
		t.Fatalf("Search(): %v", err)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })

	want := []domain.Item{
		{
			Name:           "AuroraProject",
			Path:           "/home/x/projects/aurora",
			IconPath:       "/Applications/PhpStorm.app",
			BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
			IsModernBinary: true,
			Match:          "AuroraProject aurora",
		},
		{
			Name:           "beacon",
			Path:           "/home/x/projects/beacon",
			IconPath:       "/Applications/PhpStorm.app",
			BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
			IsModernBinary: true,
			Match:          "beacon beacon",
		},
	}

	if !slices.Equal(items, want) {
		t.Errorf("Search() = %+v, want %+v", items, want)
	}
}

func TestService_Search_filtersByNameOrBasename(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(phpStormFixture()), e, Defaults())

	items, err := svc.Search(domain.PhpStorm, "AURORA")
	if err != nil {
		t.Fatalf("Search(): %v", err)
	}
	if len(items) != 1 || items[0].Name != "AuroraProject" {
		t.Errorf("Search(filter=AURORA) = %+v, want only AuroraProject", items)
	}
}

func TestService_Search_appNotFound(t *testing.T) {
	files := phpStormFixture()
	delete(files, "/Applications/PhpStorm.app/Contents/MacOS/phpstorm")
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(files), e, Defaults())

	_, err := svc.Search(domain.PhpStorm, "")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Search() error = %v, want *NotFoundError", err)
	}
}

func TestService_Search_settingsDirNotFound(t *testing.T) {
	files := map[string]string{
		"/Applications/PhpStorm.app/Contents/MacOS/phpstorm": "",
	}
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(files), e, Defaults())

	_, err := svc.Search(domain.PhpStorm, "")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Search() error = %v, want *NotFoundError", err)
	}
}

func TestService_SearchAll_skipsFailingProducts(t *testing.T) {
	files := phpStormFixture()
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	svc := NewService(iofstest.New(files), e, Defaults())

	items := svc.SearchAll("")

	if len(items) != 2 {
		t.Fatalf("SearchAll() returned %d items, want 2 (only the installed product)", len(items))
	}
	for _, item := range items {
		if item.IconPath != "/Applications/PhpStorm.app" {
			t.Errorf("SearchAll() item %+v has unexpected IconPath", item)
		}
	}
}
