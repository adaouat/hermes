package jetbrains

import (
	"errors"
	"slices"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestRepository_LocateSettingsDirectory(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		envVars map[string]string
		product domain.Product
		prefix  string
		want    string
		wantErr bool
	}{
		{
			name: "skips a version directory with only one child, picks the next",
			files: map[string]string{
				"/home/x/Library/Application Support/JetBrains/PhpStorm2024.1/options/foo.xml": "",
				"/home/x/Library/Application Support/JetBrains/PhpStorm2023.3/options/foo.xml": "",
				"/home/x/Library/Application Support/JetBrains/PhpStorm2023.3/other.txt":       "",
			},
			envVars: map[string]string{"HOME": "/home/x"},
			product: domain.PhpStorm,
			prefix:  "PhpStorm",
			want:    "/home/x/Library/Application Support/JetBrains/PhpStorm2023.3",
		},
		{
			name: "continues past a missing default path instead of erroring",
			files: map[string]string{
				"/home/x/Library/Application Support/JetBrains/PhpStorm2024.1/options/foo.xml": "",
				"/home/x/Library/Application Support/JetBrains/PhpStorm2024.1/other.txt":       "",
			},
			envVars: map[string]string{"HOME": "/home/x"},
			product: domain.PhpStorm,
			prefix:  "PhpStorm",
			want:    "/home/x/Library/Application Support/JetBrains/PhpStorm2024.1",
		},
		{
			name: "jb_settings overrides default search paths",
			files: map[string]string{
				"/custom/settings/PhpStorm2024.1/options/foo.xml": "",
				"/custom/settings/PhpStorm2024.1/other.txt":       "",
			},
			envVars: map[string]string{"jb_settings": "/custom/settings"},
			product: domain.PhpStorm,
			prefix:  "PhpStorm",
			want:    "/custom/settings/PhpStorm2024.1",
		},
		{
			name: "fleet pattern matches without a version suffix",
			files: map[string]string{
				"/home/x/Library/Application Support/JetBrains/Fleet/backend/options/foo.xml": "",
				"/home/x/Library/Application Support/JetBrains/Fleet/other.txt":               "",
			},
			envVars: map[string]string{"HOME": "/home/x"},
			product: domain.Fleet,
			prefix:  "Fleet",
			want:    "/home/x/Library/Application Support/JetBrains/Fleet",
		},
		{
			name: "android studio requires the year.quarter.fix pattern",
			files: map[string]string{
				"/home/x/Library/Application Support/Google/AndroidStudio2023.1/options/foo.xml":   "",
				"/home/x/Library/Application Support/Google/AndroidStudio2023.1/other.txt":         "",
				"/home/x/Library/Application Support/Google/AndroidStudio2024.3.2/options/foo.xml": "",
				"/home/x/Library/Application Support/Google/AndroidStudio2024.3.2/other.txt":       "",
			},
			envVars: map[string]string{"HOME": "/home/x"},
			product: domain.AndroidStudio,
			prefix:  "AndroidStudio",
			want:    "/home/x/Library/Application Support/Google/AndroidStudio2024.3.2",
		},
		{
			name:    "not found anywhere",
			envVars: map[string]string{"HOME": "/home/x"},
			product: domain.PhpStorm,
			prefix:  "PhpStorm",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRepository(iofstest.New(tc.files), envtest.New(tc.envVars), tc.product, ProductDetails{PreferencePrefix: tc.prefix})
			got, err := r.LocateSettingsDirectory()
			if tc.wantErr {
				var notFound *NotFoundError
				if !errors.As(err, &notFound) {
					t.Fatalf("LocateSettingsDirectory() error = %v, want *NotFoundError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LocateSettingsDirectory(): %v", err)
			}
			if got != tc.want {
				t.Errorf("LocateSettingsDirectory() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRepository_RecentProjects(t *testing.T) {
	const settingsDir = "/home/x/Library/Preferences/PhpStorm2024.1"

	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name: "prefers recentProjectDirectories.xml",
			files: map[string]string{
				settingsDir + "/other.txt":                            "",
				settingsDir + "/options/recentProjectDirectories.xml": `<application><component name="RecentDirectoryProjectsManager"><option name="recentPaths"><list><option value="$USER_HOME$/a" /></list></option></component></application>`,
				settingsDir + "/options/recentProjects.xml":           `<application><component name="RecentProjectsManager"><option name="recentPaths"><list><option value="$USER_HOME$/b" /></list></option></component></application>`,
			},
			want: []string{"/home/x/a"},
		},
		{
			name: "falls back to recentProjects.xml",
			files: map[string]string{
				settingsDir + "/other.txt":                  "",
				settingsDir + "/options/recentProjects.xml": `<application><component name="RecentProjectsManager"><option name="recentPaths"><list><option value="$USER_HOME$/b" /></list></option></component></application>`,
			},
			want: []string{"/home/x/b"},
		},
		{
			name: "falls back to recentSolutions.xml",
			files: map[string]string{
				settingsDir + "/other.txt":                   "",
				settingsDir + "/options/recentSolutions.xml": `<application><component name="RiderRecentProjectsManager"><option name="recentPaths"><list><option value="$USER_HOME$/c.sln" /></list></option></component></application>`,
			},
			want: []string{"/home/x/c.sln"},
		},
		{
			name: "returns nil when nothing is present and not Fleet",
			files: map[string]string{
				settingsDir + "/other.txt":   "",
				settingsDir + "/another.txt": "",
			},
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := envtest.New(map[string]string{"HOME": "/home/x"})
			r := NewRepository(iofstest.New(tc.files), e, domain.PhpStorm, ProductDetails{PreferencePrefix: "PhpStorm"})
			got, err := r.RecentProjects()
			if err != nil {
				t.Fatalf("RecentProjects(): %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("RecentProjects() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRepository_RecentProjects_fleetGlobFallback(t *testing.T) {
	const settingsDir = "/home/x/Library/Application Support/JetBrains/Fleet"
	files := map[string]string{
		settingsDir + "/other.txt":                    "",
		settingsDir + "/backend/p1/trusted-paths.xml": `<application><option name="TRUSTED_PROJECT_PATHS"><map><entry key="$USER_HOME$/trusted1" /></map></option></application>`,
	}
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	r := NewRepository(iofstest.New(files), e, domain.Fleet, ProductDetails{PreferencePrefix: "Fleet"})

	got, err := r.RecentProjects()
	if err != nil {
		t.Fatalf("RecentProjects(): %v", err)
	}
	want := []string{"/home/x/trusted1"}
	if !slices.Equal(got, want) {
		t.Errorf("RecentProjects() = %v, want %v", got, want)
	}
}

func TestRepository_RecentProjects_propagatesSettingsDirError(t *testing.T) {
	e := envtest.New(map[string]string{"HOME": "/home/x"})
	r := NewRepository(iofstest.New(nil), e, domain.PhpStorm, ProductDetails{PreferencePrefix: "PhpStorm"})

	_, err := r.RecentProjects()
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("RecentProjects() error = %v, want *NotFoundError", err)
	}
}
