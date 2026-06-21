package jetbrains

import (
	"fmt"
	"strings"

	"github.com/adaouat/hermes/pkg/domain"
)

// NotFoundError reports that a product's application bundle, binary, or settings
// directory couldn't be located, along with enough context (what was searched, where)
// for a user to troubleshoot their setup. Exposed at the package boundary so
// internal/cmd can map it to a distinct exit code (roadmap M3).
type NotFoundError struct {
	Product       domain.Product
	What          string // e.g. "application", "bin", "settings directory"
	Names         []string
	SearchedPaths []string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf(
		"jetbrains: can't locate %s for %s (looked for %s in %s)",
		e.What, e.Product.Display(), strings.Join(e.Names, ", "), strings.Join(e.SearchedPaths, ", "),
	)
}
