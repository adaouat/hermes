package jetbrains

import (
	"errors"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/internal/iofs/iofstest"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestLocator_LocateApplication(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		envVars map[string]string
		details ProductDetails
		want    string
		wantErr bool
	}{
		{
			name: "found in default /Applications",
			files: map[string]string{
				"/Applications/PhpStorm.app/Contents/Info.plist": "",
			},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}},
			want:    "/Applications/PhpStorm.app",
		},
		{
			name: "found in second default path when first is missing",
			files: map[string]string{
				"/Users/x/Applications/PhpStorm.app/Contents/Info.plist": "",
			},
			envVars: map[string]string{"HOME": "/Users/x"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}},
			want:    "/Users/x/Applications/PhpStorm.app",
		},
		{
			name: "jb_application overrides default search paths",
			files: map[string]string{
				"/Applications/PhpStorm.app/Contents/Info.plist": "",
				"/custom/apps/PhpStorm.app/Contents/Info.plist":  "",
			},
			envVars: map[string]string{"jb_application": "/custom/apps"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}},
			want:    "/custom/apps/PhpStorm.app",
		},
		{
			name: "duplicate matches: first alphabetically wins, no error",
			files: map[string]string{
				"/Applications/A.app/Contents/Info.plist": "",
				"/Applications/B.app/Contents/Info.plist": "",
			},
			details: ProductDetails{ApplicationNames: []string{"A", "B"}},
			want:    "/Applications/A.app",
		},
		{
			name:    "not found anywhere",
			details: ProductDetails{ApplicationNames: []string{"Nope"}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := NewLocator(iofstest.New(tc.files), envtest.New(tc.envVars), domain.PhpStorm, tc.details)
			got, err := l.LocateApplication()
			if tc.wantErr {
				var notFound *NotFoundError
				if !errors.As(err, &notFound) {
					t.Fatalf("LocateApplication() error = %v, want *NotFoundError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LocateApplication(): %v", err)
			}
			if got != tc.want {
				t.Errorf("LocateApplication() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLocator_LocateBin(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		envVars map[string]string
		details ProductDetails
		want    string
		wantErr bool
	}{
		{
			name: "found via PATH",
			files: map[string]string{
				"/usr/local/bin/phpstorm":                        "",
				"/Applications/PhpStorm.app/Contents/Info.plist": "",
			},
			envVars: map[string]string{"PATH": "/usr/local/bin"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}, Binaries: []string{"phpstorm", "pstorm"}},
			want:    "/usr/local/bin/phpstorm",
		},
		{
			name: "found in app's Contents/MacOS when not on PATH (post-2023 layout)",
			files: map[string]string{
				"/Applications/PhpStorm.app/Contents/MacOS/phpstorm": "",
			},
			envVars: map[string]string{"PATH": "/usr/local/bin"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}, Binaries: []string{"phpstorm", "pstorm"}},
			want:    "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
		},
		{
			name: "jb_binaries overrides PATH",
			files: map[string]string{
				"/custom/bin/phpstorm":                           "",
				"/Applications/PhpStorm.app/Contents/Info.plist": "",
			},
			envVars: map[string]string{"jb_binaries": "/custom/bin"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}, Binaries: []string{"phpstorm"}},
			want:    "/custom/bin/phpstorm",
		},
		{
			name: "duplicate matches: first alphabetically wins, no error",
			files: map[string]string{
				"/usr/local/bin/phpstorm":                        "",
				"/usr/local/bin/pstorm":                          "",
				"/Applications/PhpStorm.app/Contents/Info.plist": "",
			},
			envVars: map[string]string{"PATH": "/usr/local/bin"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}, Binaries: []string{"phpstorm", "pstorm"}},
			want:    "/usr/local/bin/phpstorm",
		},
		{
			name:    "application not found propagates the application error",
			envVars: map[string]string{"PATH": "/usr/local/bin"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}, Binaries: []string{"phpstorm"}},
			wantErr: true,
		},
		{
			name: "bin not found anywhere",
			files: map[string]string{
				"/Applications/PhpStorm.app/Contents/Info.plist": "",
			},
			envVars: map[string]string{"PATH": "/usr/local/bin"},
			details: ProductDetails{ApplicationNames: []string{"PhpStorm"}, Binaries: []string{"phpstorm"}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := NewLocator(iofstest.New(tc.files), envtest.New(tc.envVars), domain.PhpStorm, tc.details)
			got, err := l.LocateBin()
			if tc.wantErr {
				var notFound *NotFoundError
				if !errors.As(err, &notFound) {
					t.Fatalf("LocateBin() error = %v, want *NotFoundError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("LocateBin(): %v", err)
			}
			if got != tc.want {
				t.Errorf("LocateBin() = %q, want %q", got, tc.want)
			}
		})
	}
}
