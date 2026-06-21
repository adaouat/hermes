package jetbrains

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adaouat/hermes/internal/iofs"
)

// ProjectName resolves a project's human-readable name from its path.
type ProjectName struct {
	FS iofs.FS
}

// NewProjectName returns a ProjectName backed by fsys.
func NewProjectName(fsys iofs.FS) ProjectName {
	return ProjectName{FS: fsys}
}

// Resolve walks the legacy fallback chain: .idea/name, then .idea/.name, then the first
// .idea/*.iml file's basename, then a handful of workspace.xml probes, finally the
// project path's own basename (which also covers .sln solutions - the Dart source had a
// dead branch special-casing .sln that the basename fallback already handled; [smell]
// dead .sln branch).
func (n ProjectName) Resolve(projectPath string) (string, error) {
	ideaPath := filepath.Join(projectPath, ".idea")
	if n.FS.Exists(ideaPath) {
		name, ok, err := n.fromFile(filepath.Join(ideaPath, "name"))
		if err != nil {
			return "", err
		}
		if ok {
			return name, nil
		}

		name, ok, err = n.fromFile(filepath.Join(ideaPath, ".name"))
		if err != nil {
			return "", err
		}
		if ok {
			return name, nil
		}

		name, ok, err = n.fromIml(ideaPath)
		if err != nil {
			return "", err
		}
		if ok {
			return name, nil
		}

		name, ok, err = n.fromWorkspace(filepath.Join(ideaPath, "workspace.xml"))
		if err != nil {
			return "", err
		}
		if ok {
			return name, nil
		}
	}

	return basenameWithoutExt(projectPath), nil
}

func (n ProjectName) fromFile(path string) (string, bool, error) {
	if !n.FS.Exists(path) {
		return "", false, nil
	}
	content, err := n.FS.ReadFile(path)
	if err != nil {
		return "", false, fmt.Errorf("jetbrains: reading %s: %w", path, err)
	}
	return string(content), true, nil
}

func (n ProjectName) fromIml(ideaPath string) (string, bool, error) {
	entries, err := n.FS.ReadDir(ideaPath)
	if err != nil {
		return "", false, fmt.Errorf("jetbrains: reading %s: %w", ideaPath, err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".iml" {
			return basenameWithoutExt(entry.Name()), true, nil
		}
	}
	return "", false, nil
}

type workspaceXML struct {
	XMLName    xml.Name             `xml:"project"`
	Components []workspaceComponent `xml:"component"`
}

type workspaceComponent struct {
	Name    string         `xml:"name,attr"`
	Panes   []pane         `xml:"panes>pane"`
	Ignored []ignoredEntry `xml:"ignored"`
}

type pane struct {
	ID      string    `xml:"id,attr"`
	SubPane []subPane `xml:"subPane"`
}

type subPane struct {
	Path   []pathElem `xml:"PATH"`
	Expand []expand   `xml:"expand"`
}

type pathElem struct {
	Elements []pathOption `xml:"PATH_ELEMENT"`
}

type pathOption struct {
	Options []xmlOption `xml:"option"`
}

type expand struct {
	Paths []expandPath `xml:"path"`
}

type expandPath struct {
	Items []item `xml:"item"`
}

type item struct {
	Type string `xml:"type,attr"`
	Name string `xml:"name,attr"`
}

type ignoredEntry struct {
	Path string `xml:"path,attr"`
}

func (n ProjectName) fromWorkspace(path string) (string, bool, error) {
	if !n.FS.Exists(path) {
		return "", false, nil
	}
	content, err := n.FS.ReadFile(path)
	if err != nil {
		return "", false, fmt.Errorf("jetbrains: reading %s: %w", path, err)
	}

	var doc workspaceXML
	if err := xml.Unmarshal(content, &doc); err != nil {
		return "", false, fmt.Errorf("jetbrains: parsing %s: %w", path, err)
	}

	if name, ok := probePathElementOption(doc); ok {
		return name, true, nil
	}
	if name, ok := probeExpandProjectNode(doc); ok {
		return name, true, nil
	}
	if name, ok := probeIgnoredIwsPath(doc); ok {
		return name, true, nil
	}
	return "", false, nil
}

func probePathElementOption(doc workspaceXML) (string, bool) {
	for _, c := range doc.Components {
		if c.Name != "ProjectView" {
			continue
		}
		for _, p := range c.Panes {
			if p.ID != "ProjectPane" {
				continue
			}
			for _, sp := range p.SubPane {
				for _, path := range sp.Path {
					for _, elem := range path.Elements {
						for _, opt := range elem.Options {
							if opt.Value != "" {
								return opt.Value, true
							}
						}
					}
				}
			}
		}
	}
	return "", false
}

func probeExpandProjectNode(doc workspaceXML) (string, bool) {
	for _, c := range doc.Components {
		if c.Name != "ProjectView" {
			continue
		}
		for _, p := range c.Panes {
			if p.ID != "ProjectPane" {
				continue
			}
			for _, sp := range p.SubPane {
				for _, ex := range sp.Expand {
					for _, path := range ex.Paths {
						for _, it := range path.Items {
							if strings.Contains(it.Type, ":ProjectViewProjectNode") {
								return it.Name, true
							}
						}
					}
				}
			}
		}
	}
	return "", false
}

func probeIgnoredIwsPath(doc workspaceXML) (string, bool) {
	for _, c := range doc.Components {
		if c.Name != "ChangeListManager" {
			continue
		}
		for _, ig := range c.Ignored {
			if strings.Contains(ig.Path, ".iws") {
				return ig.Path, true
			}
		}
	}
	return "", false
}

func basenameWithoutExt(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
