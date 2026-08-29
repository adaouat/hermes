package jetbrains

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/adaouat/hermes/internal/env/envtest"
	"github.com/adaouat/hermes/pkg/domain"
)

func TestDefaults_count(t *testing.T) {
	if got := len(Defaults()); got != 19 {
		t.Errorf("len(Defaults()) = %d, want 19", got)
	}
}

func TestDefaults_products(t *testing.T) {
	tests := []struct {
		name                 string
		product              domain.Product
		wantApplicationNames []string
		wantPreferencePrefix string
		wantBinaries         []string
	}{
		{
			name:                 "phpStorm",
			product:              domain.PhpStorm,
			wantApplicationNames: []string{"PhpStorm"},
			wantPreferencePrefix: "PhpStorm",
			wantBinaries:         []string{"phpstorm", "pstorm"},
		},
		{
			name:                 "webStorm",
			product:              domain.WebStorm,
			wantApplicationNames: []string{"WebStorm"},
			wantPreferencePrefix: "WebStorm",
			wantBinaries:         []string{"webstorm", "wstorm"},
		},
		{
			name:                 "androidStudio",
			product:              domain.AndroidStudio,
			wantApplicationNames: []string{"Android Studio"},
			wantPreferencePrefix: "AndroidStudio",
			wantBinaries:         []string{"studio"},
		},
		{
			name:                 "fleet",
			product:              domain.Fleet,
			wantApplicationNames: []string{"Fleet"},
			wantPreferencePrefix: "Fleet",
			wantBinaries:         []string{"fleet"},
		},
		{
			name:                 "intelliJIdeaCommunity",
			product:              domain.IntelliJIdeaCommunity,
			wantApplicationNames: []string{"IntelliJ IDEA CE", "IntelliJ IDEA Community", "IntelliJ IDEA Community Edition"},
			wantPreferencePrefix: "IdeaIC",
			wantBinaries:         []string{"idea", "ideac"},
		},
		{
			name:                 "intelliJIdeaUltimate",
			product:              domain.IntelliJIdeaUltimate,
			wantApplicationNames: []string{"IntelliJ IDEA", "IntelliJ IDEA Ultimate", "IntelliJ IDEA Ultimate Edition"},
			wantPreferencePrefix: "IntelliJIdea",
			wantBinaries:         []string{"idea", "ideau"},
		},
		{
			name:                 "pyCharmProfessional",
			product:              domain.PyCharmProfessional,
			wantApplicationNames: []string{"PyCharm", "PyCharm Professional", "PyCharm Professional Edition"},
			wantPreferencePrefix: "PyCharm",
			wantBinaries:         []string{"pycharm", "charm", "pycharmp", "charmp"},
		},
		{
			name:                 "pyCharmCommunity",
			product:              domain.PyCharmCommunity,
			wantApplicationNames: []string{"PyCharm CE", "PyCharm Community", "PyCharm Community Edition"},
			wantPreferencePrefix: "PyCharmCE",
			wantBinaries:         []string{"pycharm", "charm", "pycharmc", "charmc"},
		},
		{
			name:                 "cLionNova",
			product:              domain.CLionNova,
			wantApplicationNames: []string{"CLion Nova"},
			wantPreferencePrefix: "CLionNova",
			wantBinaries:         []string{"nova", "clion"},
		},
	}

	defaults := Defaults()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			details, ok := defaults[tc.product]
			if !ok {
				t.Fatalf("Defaults() missing product %q", tc.product)
			}
			if !slices.Equal(details.ApplicationNames, tc.wantApplicationNames) {
				t.Errorf("ApplicationNames = %v, want %v", details.ApplicationNames, tc.wantApplicationNames)
			}
			if details.PreferencePrefix != tc.wantPreferencePrefix {
				t.Errorf("PreferencePrefix = %q, want %q", details.PreferencePrefix, tc.wantPreferencePrefix)
			}
			if !slices.Equal(details.Binaries, tc.wantBinaries) {
				t.Errorf("Binaries = %v, want %v", details.Binaries, tc.wantBinaries)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name                 string
		custom               map[string]any
		product              domain.Product
		wantApplicationNames []string
		wantPreferencePrefix string
		wantBinaries         []string
	}{
		{
			name:                 "no override keeps defaults",
			custom:               map[string]any{},
			product:              domain.PhpStorm,
			wantApplicationNames: []string{"PhpStorm"},
			wantPreferencePrefix: "PhpStorm",
			wantBinaries:         []string{"phpstorm", "pstorm"},
		},
		{
			name: "overrides only preferencePrefix, keeps other defaults",
			custom: map[string]any{
				"phpStorm": map[string]any{"preferencePrefix": "PhpStormCustom"},
			},
			product:              domain.PhpStorm,
			wantApplicationNames: []string{"PhpStorm"},
			wantPreferencePrefix: "PhpStormCustom",
			wantBinaries:         []string{"phpstorm", "pstorm"},
		},
		{
			name: "overrides all fields",
			custom: map[string]any{
				"phpStorm": map[string]any{
					"applicationNames": []any{"MyStorm"},
					"preferencePrefix": "MyStorm",
					"binaries":         []any{"mystorm"},
				},
			},
			product:              domain.PhpStorm,
			wantApplicationNames: []string{"MyStorm"},
			wantPreferencePrefix: "MyStorm",
			wantBinaries:         []string{"mystorm"},
		},
		{
			name: "unknown product key is ignored",
			custom: map[string]any{
				"notAProduct": map[string]any{"preferencePrefix": "Nope"},
			},
			product:              domain.PhpStorm,
			wantApplicationNames: []string{"PhpStorm"},
			wantPreferencePrefix: "PhpStorm",
			wantBinaries:         []string{"phpstorm", "pstorm"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := Merge(Defaults(), tc.custom)
			if err != nil {
				t.Fatalf("Merge: %v", err)
			}
			details, ok := merged[tc.product]
			if !ok {
				t.Fatalf("Merge() missing product %q", tc.product)
			}
			if !slices.Equal(details.ApplicationNames, tc.wantApplicationNames) {
				t.Errorf("ApplicationNames = %v, want %v", details.ApplicationNames, tc.wantApplicationNames)
			}
			if details.PreferencePrefix != tc.wantPreferencePrefix {
				t.Errorf("PreferencePrefix = %q, want %q", details.PreferencePrefix, tc.wantPreferencePrefix)
			}
			if !slices.Equal(details.Binaries, tc.wantBinaries) {
				t.Errorf("Binaries = %v, want %v", details.Binaries, tc.wantBinaries)
			}
		})
	}
}

func TestMerge_invalidFieldType(t *testing.T) {
	custom := map[string]any{
		"phpStorm": map[string]any{"binaries": "not-a-list"},
	}
	if _, err := Merge(Defaults(), custom); err == nil {
		t.Error("Merge() error = nil, want error for invalid binaries type")
	}
}

func TestLoad_noCustomConfig(t *testing.T) {
	e := envtest.New(nil)
	cfg, err := Load(e)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg) != 19 {
		t.Errorf("len(Load()) = %d, want 19", len(cfg))
	}
	if cfg[domain.PhpStorm].PreferencePrefix != "PhpStorm" {
		t.Errorf("PreferencePrefix = %q, want %q", cfg[domain.PhpStorm].PreferencePrefix, "PhpStorm")
	}
}

func TestLoad_withCustomConfig(t *testing.T) {
	e := envtest.New(map[string]string{
		"jb_custom_config": `{"phpStorm": {"preferencePrefix": "Custom"}}`,
	})
	cfg, err := Load(e)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg[domain.PhpStorm].PreferencePrefix; got != "Custom" {
		t.Errorf("PreferencePrefix = %q, want %q", got, "Custom")
	}
}

func TestLoad_invalidJSON(t *testing.T) {
	e := envtest.New(map[string]string{
		"jb_custom_config": `{not json`,
	})
	if _, err := Load(e); err == nil {
		t.Error("Load() error = nil, want error for invalid JSON")
	}
}

func TestProductDetails_marshalKeys(t *testing.T) {
	raw, err := json.Marshal(Defaults()[domain.PhpStorm])
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	for _, key := range []string{"applicationNames", "preferencePrefix", "binaries"} {
		if _, ok := got[key]; !ok {
			t.Errorf("marshaled ProductDetails missing key %q, got %v", key, got)
		}
	}
}

func TestConfig_marshalRoundTrips(t *testing.T) {
	raw, err := json.Marshal(Defaults())
	if err != nil {
		t.Fatalf("Marshal(Defaults()): %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}

	merged, err := Merge(Defaults(), decoded)
	if err != nil {
		t.Fatalf("Merge(Defaults(), decoded): %v", err)
	}
	if !reflect.DeepEqual(merged, Defaults()) {
		t.Errorf("round-tripped config != Defaults()")
	}
}
