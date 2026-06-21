package jetbrains

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type xmlApplication struct {
	XMLName    xml.Name       `xml:"application"`
	Components []xmlComponent `xml:"component"`
	Options    []xmlOption    `xml:"option"`
}

type xmlComponent struct {
	Name    string      `xml:"name,attr"`
	Options []xmlOption `xml:"option"`
}

type xmlOption struct {
	Name  string   `xml:"name,attr"`
	Value string   `xml:"value,attr"`
	List  *xmlList `xml:"list"`
	Map   *xmlMap  `xml:"map"`
}

type xmlList struct {
	Options []xmlOption `xml:"option"`
}

type xmlMap struct {
	Entries []xmlEntry `xml:"entry"`
}

type xmlEntry struct {
	Key string `xml:"key,attr"`
}

func parseApplication(content string) (xmlApplication, error) {
	var doc xmlApplication
	if err := xml.Unmarshal([]byte(content), &doc); err != nil {
		return xmlApplication{}, fmt.Errorf("jetbrains: parsing recent projects xml: %w", err)
	}
	return doc, nil
}

func findComponent(components []xmlComponent, name string) (xmlComponent, bool) {
	for _, c := range components {
		if c.Name == name {
			return c, true
		}
	}
	return xmlComponent{}, false
}

func findOption(options []xmlOption, name string) (xmlOption, bool) {
	for _, o := range options {
		if o.Name == name {
			return o, true
		}
	}
	return xmlOption{}, false
}

func expandHome(value, home string) string {
	return strings.ReplaceAll(value, "$USER_HOME$", home)
}

// additionalInfoOrRecentPaths implements the two-branch lookup shared by
// ExtractRecentProjects and ExtractRecentSolutions: prefer additionalInfo/map/entry/@key,
// falling back to recentPaths/list/option/@value.
func additionalInfoOrRecentPaths(options []xmlOption, home string) []string {
	if additionalInfo, ok := findOption(options, "additionalInfo"); ok && additionalInfo.Map != nil {
		var keys []string
		for _, entry := range additionalInfo.Map.Entries {
			if entry.Key != "" {
				keys = append(keys, expandHome(entry.Key, home))
			}
		}
		if len(keys) > 0 {
			return keys
		}
	}

	if recentPaths, ok := findOption(options, "recentPaths"); ok && recentPaths.List != nil {
		var values []string
		for _, opt := range recentPaths.List.Options {
			if opt.Value != "" {
				values = append(values, expandHome(opt.Value, home))
			}
		}
		return values
	}

	return nil
}

// ExtractRecentProjects parses a recentProjects.xml document.
func ExtractRecentProjects(content, home string) ([]string, error) {
	doc, err := parseApplication(content)
	if err != nil {
		return nil, err
	}
	component, ok := findComponent(doc.Components, "RecentProjectsManager")
	if !ok {
		return nil, nil
	}
	return additionalInfoOrRecentPaths(component.Options, home), nil
}

// ExtractRecentProjectDirectories parses a recentProjectDirectories.xml document.
func ExtractRecentProjectDirectories(content, home string) ([]string, error) {
	doc, err := parseApplication(content)
	if err != nil {
		return nil, err
	}
	component, ok := findComponent(doc.Components, "RecentDirectoryProjectsManager")
	if !ok {
		return nil, nil
	}
	recentPaths, ok := findOption(component.Options, "recentPaths")
	if !ok || recentPaths.List == nil {
		return nil, nil
	}
	var values []string
	for _, opt := range recentPaths.List.Options {
		if opt.Value != "" {
			values = append(values, expandHome(opt.Value, home))
		}
	}
	return values, nil
}

// ExtractRecentSolutions parses a recentSolutions.xml document (Rider).
func ExtractRecentSolutions(content, home string) ([]string, error) {
	doc, err := parseApplication(content)
	if err != nil {
		return nil, err
	}
	component, ok := findComponent(doc.Components, "RiderRecentProjectsManager")
	if !ok {
		return nil, nil
	}
	return additionalInfoOrRecentPaths(component.Options, home), nil
}

// ExtractTrustedPaths parses a Fleet trusted-paths.xml document.
func ExtractTrustedPaths(content, home string) ([]string, error) {
	doc, err := parseApplication(content)
	if err != nil {
		return nil, err
	}
	option, ok := findOption(doc.Options, "TRUSTED_PROJECT_PATHS")
	if !ok || option.Map == nil {
		return nil, nil
	}
	var keys []string
	for _, entry := range option.Map.Entries {
		if entry.Key != "" {
			keys = append(keys, expandHome(entry.Key, home))
		}
	}
	return keys, nil
}
