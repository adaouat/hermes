package domain

// Item is a launcher-neutral recent-project entry. Launcher adapters
// (internal/launcher/*) translate it into their own output format; no JSON tags live
// here since this type carries no opinion about serialization.
type Item struct {
	Name           string
	Path           string
	IconPath       string
	BinaryPath     string
	IsModernBinary bool
	Match          string
}
