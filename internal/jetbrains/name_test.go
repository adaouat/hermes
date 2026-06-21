package jetbrains

import (
	"testing"

	"github.com/adaouat/hermes/internal/iofs/iofstest"
)

func TestProjectName_Resolve(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		files       map[string]string
		want        string
	}{
		{
			name:        "returns basename when no .idea directory",
			projectPath: "/tmp/x/my-project",
			files:       map[string]string{"/tmp/x/my-project/README.md": ""},
			want:        "my-project",
		},
		{
			name:        "reads name from .idea/name file",
			projectPath: "/tmp/x/project-dir",
			files:       map[string]string{"/tmp/x/project-dir/.idea/name": "CustomName"},
			want:        "CustomName",
		},
		{
			name:        "reads name from .idea/.name file",
			projectPath: "/tmp/x/project-dir2",
			files:       map[string]string{"/tmp/x/project-dir2/.idea/.name": "DotName"},
			want:        "DotName",
		},
		{
			name:        "prefers name file over .name file",
			projectPath: "/tmp/x/project-dir4",
			files: map[string]string{
				"/tmp/x/project-dir4/.idea/name":  "NameFile",
				"/tmp/x/project-dir4/.idea/.name": "DotNameFile",
			},
			want: "NameFile",
		},
		{
			name:        "reads name from .iml file",
			projectPath: "/tmp/x/project-dir3",
			files:       map[string]string{"/tmp/x/project-dir3/.idea/my-module.iml": ""},
			want:        "my-module",
		},
		{
			name:        "handles .sln extension fallback",
			projectPath: "/tmp/x/MySolution.sln",
			files:       map[string]string{"/tmp/x/other.txt": ""},
			want:        "MySolution",
		},
		{
			name:        "reads name from workspace.xml PATH_ELEMENT option (probe 1)",
			projectPath: "/tmp/x/project-dir5",
			files: map[string]string{
				"/tmp/x/project-dir5/.idea/workspace.xml": `<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectView">
    <panes>
      <pane id="ProjectPane">
        <subPane>
          <PATH>
            <PATH_ELEMENT>
              <option value="WorkspaceProbeName" />
            </PATH_ELEMENT>
          </PATH>
        </subPane>
      </pane>
    </panes>
  </component>
</project>`,
			},
			want: "WorkspaceProbeName",
		},
		{
			name:        "reads name from workspace.xml expand item (probe 2)",
			projectPath: "/tmp/x/project-dir6",
			files: map[string]string{
				"/tmp/x/project-dir6/.idea/workspace.xml": `<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectView">
    <panes>
      <pane id="ProjectPane">
        <subPane>
          <expand>
            <path>
              <item name="ExpandProbeName" type="com.intellij.ide.projectView.impl.nodes.NamedLibraryElementNode:ProjectViewProjectNode" />
            </path>
          </expand>
        </subPane>
      </pane>
    </panes>
  </component>
</project>`,
			},
			want: "ExpandProbeName",
		},
		{
			name:        "reads path from workspace.xml ChangeListManager ignored entry (probe 3)",
			projectPath: "/tmp/x/project-dir7",
			files: map[string]string{
				"/tmp/x/project-dir7/.idea/workspace.xml": `<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ChangeListManager">
    <ignored path="$PROJECT_DIR$/IgnoredProbe.iws" />
  </component>
</project>`,
			},
			want: "$PROJECT_DIR$/IgnoredProbe.iws",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n := NewProjectName(iofstest.New(tc.files))
			got, err := n.Resolve(tc.projectPath)
			if err != nil {
				t.Fatalf("Resolve(): %v", err)
			}
			if got != tc.want {
				t.Errorf("Resolve() = %q, want %q", got, tc.want)
			}
		})
	}
}
