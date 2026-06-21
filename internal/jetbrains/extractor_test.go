package jetbrains

import (
	"slices"
	"sort"
	"testing"
)

func TestExtractRecentProjects(t *testing.T) {
	tests := []struct {
		name string
		xml  string
		want []string
	}{
		{
			name: "extracts from additionalInfo map",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<application>
  <component name="RecentProjectsManager">
    <option name="additionalInfo">
      <map>
        <entry key="$USER_HOME$/projects/project1" />
        <entry key="$USER_HOME$/projects/project2" />
      </map>
    </option>
  </component>
</application>`,
			want: []string{"/home/x/projects/project1", "/home/x/projects/project2"},
		},
		{
			name: "extracts from recentPaths list when additionalInfo is absent",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<application>
  <component name="RecentProjectsManager">
    <option name="recentPaths">
      <list>
        <option value="$USER_HOME$/dev/app1" />
        <option value="$USER_HOME$/dev/app2" />
      </list>
    </option>
  </component>
</application>`,
			want: []string{"/home/x/dev/app1", "/home/x/dev/app2"},
		},
		{
			name: "no RecentProjectsManager component returns empty",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<application>
  <component name="OtherComponent"></component>
</application>`,
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractRecentProjects(tc.xml, "/home/x")
			if err != nil {
				t.Fatalf("ExtractRecentProjects: %v", err)
			}
			sort.Strings(got)
			want := slices.Clone(tc.want)
			sort.Strings(want)
			if !slices.Equal(got, want) {
				t.Errorf("ExtractRecentProjects() = %v, want %v", got, want)
			}
		})
	}
}

func TestExtractRecentProjectDirectories(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<application>
  <component name="RecentDirectoryProjectsManager">
    <option name="recentPaths">
      <list>
        <option value="$USER_HOME$/workspace/proj" />
      </list>
    </option>
  </component>
</application>`

	got, err := ExtractRecentProjectDirectories(xmlContent, "/home/x")
	if err != nil {
		t.Fatalf("ExtractRecentProjectDirectories: %v", err)
	}
	want := []string{"/home/x/workspace/proj"}
	if !slices.Equal(got, want) {
		t.Errorf("ExtractRecentProjectDirectories() = %v, want %v", got, want)
	}
}

func TestExtractRecentSolutions(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<application>
  <component name="RiderRecentProjectsManager">
    <option name="additionalInfo">
      <map>
        <entry key="$USER_HOME$/solutions/MySolution.sln" />
      </map>
    </option>
  </component>
</application>`

	got, err := ExtractRecentSolutions(xmlContent, "/home/x")
	if err != nil {
		t.Fatalf("ExtractRecentSolutions: %v", err)
	}
	want := []string{"/home/x/solutions/MySolution.sln"}
	if !slices.Equal(got, want) {
		t.Errorf("ExtractRecentSolutions() = %v, want %v", got, want)
	}
}

func TestExtractTrustedPaths(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<application>
  <option name="TRUSTED_PROJECT_PATHS">
    <map>
      <entry key="$USER_HOME$/trusted/project1" />
      <entry key="$USER_HOME$/trusted/project2" />
    </map>
  </option>
</application>`

	got, err := ExtractTrustedPaths(xmlContent, "/home/x")
	if err != nil {
		t.Fatalf("ExtractTrustedPaths: %v", err)
	}
	sort.Strings(got)
	want := []string{"/home/x/trusted/project1", "/home/x/trusted/project2"}
	if !slices.Equal(got, want) {
		t.Errorf("ExtractTrustedPaths() = %v, want %v", got, want)
	}
}

func TestExtract_invalidXML(t *testing.T) {
	if _, err := ExtractRecentProjects("not xml", "/home/x"); err == nil {
		t.Error("ExtractRecentProjects() error = nil, want error for invalid XML")
	}
}
