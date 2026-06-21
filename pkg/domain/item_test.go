package domain

import "testing"

func TestItem_zeroValue(t *testing.T) {
	var item Item
	if item.Name != "" || item.Path != "" || item.IconPath != "" || item.BinaryPath != "" || item.Match != "" {
		t.Errorf("zero Item has non-empty string field: %+v", item)
	}
	if item.IsModernBinary {
		t.Errorf("zero Item.IsModernBinary = true, want false")
	}
}

func TestItem_fields(t *testing.T) {
	item := Item{
		Name:           "my-project",
		Path:           "/Users/x/projects/my-project",
		IconPath:       "/Applications/PhpStorm.app",
		BinaryPath:     "/opt/homebrew/bin/phpstorm",
		IsModernBinary: true,
		Match:          "my-project /Users/x/projects/my-project",
	}

	if item.Name != "my-project" {
		t.Errorf("Name = %q, want %q", item.Name, "my-project")
	}
	if item.Path != "/Users/x/projects/my-project" {
		t.Errorf("Path = %q, want %q", item.Path, "/Users/x/projects/my-project")
	}
	if item.IconPath != "/Applications/PhpStorm.app" {
		t.Errorf("IconPath = %q, want %q", item.IconPath, "/Applications/PhpStorm.app")
	}
	if item.BinaryPath != "/opt/homebrew/bin/phpstorm" {
		t.Errorf("BinaryPath = %q, want %q", item.BinaryPath, "/opt/homebrew/bin/phpstorm")
	}
	if !item.IsModernBinary {
		t.Errorf("IsModernBinary = false, want true")
	}
	if item.Match != "my-project /Users/x/projects/my-project" {
		t.Errorf("Match = %q, want %q", item.Match, "my-project /Users/x/projects/my-project")
	}
}
