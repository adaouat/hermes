package raycast

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/adaouat/hermes/pkg/domain"
)

// envelope is the {"items": [...]} shape the Raycast extension consumes (docs/adr/0005).
type envelope struct {
	Items []item `json:"items"`
}

// item mirrors one domain.Item as a Raycast list entry with its open action.
type item struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle"`
	Icon     string   `json:"icon"`
	Actions  []action `json:"actions"`
}

// action describes how the extension invokes the underlying JetBrains binary. isModernBinary
// mirrors the branch the Alfred workflow's own open script takes
// (internal/launcher/alfred/assets/info.plist.tmpl): a modern (post-2023 DMG layout) binary
// is launched via `open -na <binary> --args <path>`, a legacy one is exec'd directly with
// path as its sole argument. The extension needs this to invoke the right one.
type action struct {
	Type           string `json:"type"`
	Binary         string `json:"binary"`
	Path           string `json:"path"`
	IsModernBinary bool   `json:"isModernBinary"`
}

// buildItem returns it as a Raycast item. Actions is empty (not omitted - always a JSON
// array) when it has no located binary: there is nothing to open.
func buildItem(it domain.Item) item {
	actions := make([]action, 0, 1)
	if it.BinaryPath != "" {
		actions = append(actions, action{
			Type:           "open",
			Binary:         it.BinaryPath,
			Path:           it.Path,
			IsModernBinary: it.IsModernBinary,
		})
	}
	return item{
		ID:       it.Name,
		Title:    it.Name,
		Subtitle: it.Path,
		Icon:     it.IconPath,
		Actions:  actions,
	}
}

func writeEnvelope(w io.Writer, items []domain.Item) error {
	out := make([]item, 0, len(items))
	for _, it := range items {
		out = append(out, buildItem(it))
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(envelope{Items: out}); err != nil {
		return fmt.Errorf("encoding raycast envelope: %w", err)
	}
	return nil
}
