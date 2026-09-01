package raycast

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/adaouat/hermes/pkg/domain"
)

//go:embed testdata/search_basic.json
var goldenSearchBasic []byte

func TestBuildItem(t *testing.T) {
	tests := []struct {
		name string
		item domain.Item
		want item
	}{
		{
			name: "modern binary gets an open action",
			item: domain.Item{
				Name:           "AuroraProject",
				Path:           "/Users/x/projects/aurora",
				IconPath:       "/Applications/PhpStorm.app",
				BinaryPath:     "/Applications/PhpStorm.app/Contents/MacOS/phpstorm",
				IsModernBinary: true,
				Match:          "AuroraProject aurora",
			},
			want: item{
				ID: "AuroraProject", Title: "AuroraProject",
				Subtitle: "/Users/x/projects/aurora", Icon: "/Applications/PhpStorm.app",
				Actions: []action{
					{Type: "open", Binary: "/Applications/PhpStorm.app/Contents/MacOS/phpstorm", Path: "/Users/x/projects/aurora", IsModernBinary: true},
				},
			},
		},
		{
			name: "legacy binary open action has isModernBinary false",
			item: domain.Item{
				Name: "n", Path: "/p", IconPath: "/x/icon.icns", BinaryPath: "/x/bin/phpstorm", Match: "n p",
			},
			want: item{
				ID: "n", Title: "n", Subtitle: "/p", Icon: "/x/icon.icns",
				Actions: []action{{Type: "open", Binary: "/x/bin/phpstorm", Path: "/p", IsModernBinary: false}},
			},
		},
		{
			name: "empty binary path omits actions",
			item: domain.Item{Name: "n", Path: "/p", IconPath: "/a.app", Match: "n p"},
			want: item{ID: "n", Title: "n", Subtitle: "/p", Icon: "/a.app", Actions: []action{}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildItem(tc.item)
			if got.ID != tc.want.ID || got.Title != tc.want.Title || got.Subtitle != tc.want.Subtitle || got.Icon != tc.want.Icon {
				t.Errorf("buildItem() = %+v, want %+v", got, tc.want)
			}
			if len(got.Actions) != len(tc.want.Actions) {
				t.Fatalf("buildItem().Actions = %+v, want %+v", got.Actions, tc.want.Actions)
			}
			for i := range got.Actions {
				if got.Actions[i] != tc.want.Actions[i] {
					t.Errorf("buildItem().Actions[%d] = %+v, want %+v", i, got.Actions[i], tc.want.Actions[i])
				}
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

	var buf bytes.Buffer
	if err := writeEnvelope(&buf, []domain.Item{item}); err != nil {
		t.Fatalf("writeEnvelope(): %v", err)
	}

	if buf.String() != string(goldenSearchBasic) {
		t.Errorf("writeEnvelope() output mismatch:\ngot:  %s\nwant: %s", buf.String(), goldenSearchBasic)
	}
}

func TestWriteEnvelope_emptyItemsIsEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	if err := writeEnvelope(&buf, nil); err != nil {
		t.Fatalf("writeEnvelope(): %v", err)
	}
	if buf.String() != `{"items":[]}`+"\n" {
		t.Errorf("writeEnvelope(nil) = %q, want %q", buf.String(), `{"items":[]}`+"\n")
	}
}
