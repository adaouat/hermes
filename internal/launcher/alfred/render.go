package alfred

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/adaouat/hermes/pkg/domain"
)

// resultItem mirrors the legacy CLI's ResultItem (alfred/result_item.dart)
// field-for-field, including declaration order, so json.Marshal's key order
// stays byte-identical to the Dart json_serializable output (docs/adr/0004).
type resultItem struct {
	UID          string               `json:"uid"`
	Title        string               `json:"title"`
	Match        string               `json:"match"`
	Subtitle     string               `json:"subtitle"`
	Arg          string               `json:"arg"`
	Autocomplete string               `json:"autocomplete"`
	Text         resultItemText       `json:"text"`
	Icon         resultItemIcon       `json:"icon"`
	Variables    *resultItemVariables `json:"variables,omitempty"`
}

type resultItemText struct {
	Copy      string `json:"copy"`
	LargeType string `json:"largetype"`
}

type resultItemIcon struct {
	Path string `json:"path"`
	Type string `json:"type,omitempty"`
}

type resultItemVariables struct {
	ProjectName    string `json:"jb_project_name"`
	Bin            string `json:"jb_bin"`
	SearchBasename string `json:"jb_search_basename"`
	IsNewBin       bool   `json:"jb_is_new_bin"`
}

// envelope is the {"cache": ..., "items": [...]} shape every render uses.
// Unlike the legacy CLI (whose renderItem skipped this envelope — [bug]
// renderItem vs renderItems shape mismatch), there is only one render path
// here, so every caller gets the same shape.
type envelope struct {
	Cache cacheInfo    `json:"cache"`
	Items []resultItem `json:"items"`
}

type cacheInfo struct {
	Seconds     int  `json:"seconds"`
	LooseReload bool `json:"loosereload"`
}

func buildResultItem(item domain.Item) resultItem {
	iconType := "fileicon"
	if filepath.Ext(item.IconPath) == ".icns" {
		iconType = ""
	}

	var variables *resultItemVariables
	if item.BinaryPath != "" {
		variables = &resultItemVariables{
			ProjectName:    item.Name,
			Bin:            item.BinaryPath,
			SearchBasename: filepath.Base(item.Path),
			IsNewBin:       item.IsModernBinary,
		}
	}

	return resultItem{
		UID:          item.Name,
		Title:        item.Name,
		Match:        item.Match,
		Subtitle:     item.Path,
		Arg:          item.Path,
		Autocomplete: item.Name,
		Text:         resultItemText{Copy: item.Path, LargeType: item.Name},
		Icon:         resultItemIcon{Path: item.IconPath, Type: iconType},
		Variables:    variables,
	}
}

func writeEnvelope(w io.Writer, env envelope, pretty bool) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("encoding alfred envelope: %w", err)
	}
	return nil
}
