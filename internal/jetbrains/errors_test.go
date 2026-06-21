package jetbrains

import (
	"strings"
	"testing"

	"github.com/adaouat/hermes/pkg/domain"
)

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		Product:       domain.PhpStorm,
		What:          "application",
		Names:         []string{"PhpStorm"},
		SearchedPaths: []string{"/Applications", "~/Applications"},
	}

	got := err.Error()
	if !strings.Contains(got, "PhpStorm") {
		t.Errorf("Error() = %q, want it to contain the product's Display name", got)
	}
	if !strings.Contains(got, "application") {
		t.Errorf("Error() = %q, want it to contain what was being located", got)
	}
}
