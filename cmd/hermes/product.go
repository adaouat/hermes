package main

import (
	"fmt"
	"slices"

	"github.com/adaouat/hermes/pkg/domain"
)

// parseProduct validates raw against every domain.Product hermes supports. An unclassified
// error like this one defaults to exitcode.Usage via forge's exitcode.Resolve.
func parseProduct(raw string) (domain.Product, error) {
	p := domain.Product(raw)
	if !slices.Contains(domain.Products(), p) {
		return "", fmt.Errorf(`unknown product %q (run "hermes configuration" for the full list)`, raw)
	}
	return p, nil
}
