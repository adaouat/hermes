package alfred

import (
	"bytes"
	"os"
	"testing"

	"github.com/adaouat/hermes/pkg/domain"
)

func TestBuildResultItem(t *testing.T) {
	tests := []struct {
		name string
		item domain.Item
		want resultItem
	}{
		{
			name: "modern binary gets fileicon and variables",
			item: domain.Item{
				Name:           "AuroraProject",
				Path:           "/Users/x/projects/aurora",
				IconPath:       "/Applications/PhpStorm.app",
				BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
				IsModernBinary: true,
				Match:          "AuroraProject aurora",
			},
			want: resultItem{
				UID: "AuroraProject", Title: "AuroraProject", Match: "AuroraProject aurora",
				Subtitle: "/Users/x/projects/aurora", Arg: "/Users/x/projects/aurora", Autocomplete: "AuroraProject",
				Text: resultItemText{Copy: "/Users/x/projects/aurora", LargeType: "AuroraProject"},
				Icon: resultItemIcon{Path: "/Applications/PhpStorm.app", Type: "fileicon"},
				Variables: &resultItemVariables{
					ProjectName: "AuroraProject", Bin: "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
					SearchBasename: "aurora", IsNewBin: true,
				},
			},
		},
		{
			name: "icns icon path has no type",
			item: domain.Item{Name: "n", Path: "/p", IconPath: "/x/icon.icns", Match: "n p"},
			want: resultItem{
				UID: "n", Title: "n", Match: "n p", Subtitle: "/p", Arg: "/p", Autocomplete: "n",
				Text: resultItemText{Copy: "/p", LargeType: "n"},
				Icon: resultItemIcon{Path: "/x/icon.icns"},
			},
		},
		{
			name: "empty binary path omits variables",
			item: domain.Item{Name: "n", Path: "/p", IconPath: "/a.app", Match: "n p"},
			want: resultItem{
				UID: "n", Title: "n", Match: "n p", Subtitle: "/p", Arg: "/p", Autocomplete: "n",
				Text: resultItemText{Copy: "/p", LargeType: "n"},
				Icon: resultItemIcon{Path: "/a.app", Type: "fileicon"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildResultItem(tc.item)
			if got.UID != tc.want.UID || got.Title != tc.want.Title || got.Match != tc.want.Match ||
				got.Subtitle != tc.want.Subtitle || got.Arg != tc.want.Arg || got.Autocomplete != tc.want.Autocomplete ||
				got.Text != tc.want.Text || got.Icon != tc.want.Icon {
				t.Errorf("buildResultItem() = %+v, want %+v", got, tc.want)
			}
			switch {
			case tc.want.Variables == nil && got.Variables != nil:
				t.Errorf("Variables = %+v, want nil", got.Variables)
			case tc.want.Variables != nil && (got.Variables == nil || *got.Variables != *tc.want.Variables):
				t.Errorf("Variables = %+v, want %+v", got.Variables, tc.want.Variables)
			}
		})
	}
}

func TestWriteEnvelope_goldenSearchBasic(t *testing.T) {
	item := domain.Item{
		Name:           "AuroraProject",
		Path:           "/Users/x/projects/aurora",
		IconPath:       "/Applications/PhpStorm.app",
		BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
		IsModernBinary: true,
		Match:          "AuroraProject aurora",
	}

	want, err := os.ReadFile("../../../test/fixtures/alfred/search_basic.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var buf bytes.Buffer
	env := envelope{Cache: cacheInfo{Seconds: 86400, LooseReload: true}, Items: []resultItem{buildResultItem(item)}}
	if err := writeEnvelope(&buf, env, false); err != nil {
		t.Fatalf("writeEnvelope(): %v", err)
	}

	if buf.String() != string(want) {
		t.Errorf("writeEnvelope() output mismatch:\ngot:  %s\nwant: %s", buf.String(), want)
	}
}
