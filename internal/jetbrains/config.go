package jetbrains

import (
	"encoding/json"
	"fmt"

	"github.com/adaouat/hermes/internal/env"
	"github.com/adaouat/hermes/pkg/domain"
)

// ProductDetails carries the per-product defaults needed to locate an installed
// JetBrains IDE: its application bundle name(s), settings-directory prefix, and binary
// name(s).
type ProductDetails struct {
	ApplicationNames []string
	PreferencePrefix string
	Binaries         []string
}

// Config maps every supported Product to its ProductDetails.
type Config map[domain.Product]ProductDetails

// Defaults returns the built-in configuration for every supported product, ported
// directly from the legacy Dart CLI's product_config.dart.
func Defaults() Config {
	return Config{
		domain.AndroidStudio: {
			ApplicationNames: []string{"Android Studio"},
			PreferencePrefix: "AndroidStudio",
			Binaries:         []string{"studio"},
		},
		domain.AppCode: {
			ApplicationNames: []string{"AppCode"},
			PreferencePrefix: "AppCode",
			Binaries:         []string{"appcode"},
		},
		domain.Aqua: {
			ApplicationNames: []string{"Aqua"},
			PreferencePrefix: "Aqua",
			Binaries:         []string{"aqua"},
		},
		domain.CLion: {
			ApplicationNames: []string{"CLion"},
			PreferencePrefix: "CLion",
			Binaries:         []string{"clion"},
		},
		domain.CLionNova: {
			ApplicationNames: []string{"CLion Nova"},
			PreferencePrefix: "CLionNova",
			Binaries:         []string{"nova", "clion"},
		},
		domain.DataGrip: {
			ApplicationNames: []string{"DataGrip"},
			PreferencePrefix: "DataGrip",
			Binaries:         []string{"datagrip"},
		},
		domain.DataSpell: {
			ApplicationNames: []string{"DataSpell"},
			PreferencePrefix: "DataSpell",
			Binaries:         []string{"dataspell"},
		},
		domain.Fleet: {
			ApplicationNames: []string{"Fleet"},
			PreferencePrefix: "Fleet",
			Binaries:         []string{"fleet"},
		},
		domain.GoLand: {
			ApplicationNames: []string{"GoLand"},
			PreferencePrefix: "GoLand",
			Binaries:         []string{"goland"},
		},
		domain.IntelliJIdeaCommunity: {
			ApplicationNames: []string{
				"IntelliJ IDEA CE",
				"IntelliJ IDEA Community",
				"IntelliJ IDEA Community Edition",
			},
			PreferencePrefix: "IdeaIC",
			Binaries:         []string{"idea", "ideac"},
		},
		domain.IntelliJIdeaUltimate: {
			ApplicationNames: []string{
				"IntelliJ IDEA",
				"IntelliJ IDEA Ultimate",
				"IntelliJ IDEA Ultimate Edition",
			},
			PreferencePrefix: "IntelliJIdea",
			Binaries:         []string{"idea", "ideau"},
		},
		domain.PhpStorm: {
			ApplicationNames: []string{"PhpStorm"},
			PreferencePrefix: "PhpStorm",
			Binaries:         []string{"phpstorm", "pstorm"},
		},
		domain.PyCharmProfessional: {
			ApplicationNames: []string{
				"PyCharm",
				"PyCharm Professional",
				"PyCharm Professional Edition",
			},
			PreferencePrefix: "PyCharm",
			Binaries:         []string{"pycharm", "charm", "pycharmp", "charmp"},
		},
		domain.PyCharmCommunity: {
			ApplicationNames: []string{
				"PyCharm CE",
				"PyCharm Community",
				"PyCharm Community Edition",
			},
			PreferencePrefix: "PyCharmCE",
			Binaries:         []string{"pycharm", "charm", "pycharmc", "charmc"},
		},
		domain.Rider: {
			ApplicationNames: []string{"Rider"},
			PreferencePrefix: "Rider",
			Binaries:         []string{"rider"},
		},
		domain.RubyMine: {
			ApplicationNames: []string{"RubyMine"},
			PreferencePrefix: "RubyMine",
			Binaries:         []string{"rubymine", "mine"},
		},
		domain.RustRover: {
			ApplicationNames: []string{"RustRover"},
			PreferencePrefix: "RustRover",
			Binaries:         []string{"rustrover"},
		},
		domain.WebStorm: {
			ApplicationNames: []string{"WebStorm"},
			PreferencePrefix: "WebStorm",
			Binaries:         []string{"webstorm", "wstorm"},
		},
		domain.Writerside: {
			ApplicationNames: []string{"Writerside"},
			PreferencePrefix: "Writerside",
			Binaries:         []string{"writerside"},
		},
	}
}

// Merge applies per-product overrides from custom (decoded from the jb_custom_config
// JSON env var) on top of defaults. Only products already present in defaults can be
// overridden; unknown keys in custom are ignored. Within a product's override, only the
// fields actually present override the default - missing fields keep their default
// value.
func Merge(defaults Config, custom map[string]any) (Config, error) {
	merged := make(Config, len(defaults))
	for product, details := range defaults {
		override, ok := custom[string(product)]
		if !ok {
			merged[product] = details
			continue
		}

		overrideMap, ok := override.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("jetbrains: override for product %q must be an object", product)
		}

		result, err := mergeOne(details, overrideMap)
		if err != nil {
			return nil, fmt.Errorf("jetbrains: override for product %q: %w", product, err)
		}
		merged[product] = result
	}
	return merged, nil
}

func mergeOne(details ProductDetails, override map[string]any) (ProductDetails, error) {
	result := details
	if err := overrideStrings(override, "applicationNames", &result.ApplicationNames); err != nil {
		return ProductDetails{}, err
	}
	if err := overrideString(override, "preferencePrefix", &result.PreferencePrefix); err != nil {
		return ProductDetails{}, err
	}
	if err := overrideStrings(override, "binaries", &result.Binaries); err != nil {
		return ProductDetails{}, err
	}
	return result, nil
}

func overrideString(m map[string]any, key string, target *string) error {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("field %q must be a string", key)
	}
	*target = s
	return nil
}

func overrideStrings(m map[string]any, key string, target *[]string) error {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		return fmt.Errorf("field %q must be a list of strings", key)
	}
	strs := make([]string, len(list))
	for i, item := range list {
		s, ok := item.(string)
		if !ok {
			return fmt.Errorf("field %q must be a list of strings", key)
		}
		strs[i] = s
	}
	*target = strs
	return nil
}

// Load reads jb_custom_config from e (if set) and merges it onto Defaults().
func Load(e env.Env) (Config, error) {
	raw, ok := e.Lookup("jb_custom_config")
	if !ok || raw == "" {
		return Defaults(), nil
	}

	var custom map[string]any
	if err := json.Unmarshal([]byte(raw), &custom); err != nil {
		return nil, fmt.Errorf("jetbrains: parsing jb_custom_config: %w", err)
	}

	return Merge(Defaults(), custom)
}
