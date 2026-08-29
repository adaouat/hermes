package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/jetbrains"
)

func newSearchCmd(rt *runtime) *cobra.Command {
	var product, filter string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search recent projects for one JetBrains product",
		RunE: func(c *cobra.Command, _ []string) error {
			return runSearch(c.OutOrStdout(), rt, product, filter)
		},
	}
	cmd.Flags().StringVar(&product, "product", "", `JetBrains product (see "hermes configuration" for the full list)`)
	cmd.Flags().StringVar(&filter, "filter", "", "Filter projects by name or path")
	_ = cmd.MarkFlagRequired("product")
	return cmd
}

func runSearch(w io.Writer, rt *runtime, productFlag, filter string) error {
	product, err := parseProduct(productFlag)
	if err != nil {
		return err
	}

	svc := jetbrains.NewService(rt.fs, rt.env, rt.config)
	items, err := svc.Search(product, filter)
	if err != nil {
		var notFound *jetbrains.NotFoundError
		if errors.As(err, &notFound) {
			return forgeexit.Wrap(exitNotFound, err)
		}
		return err
	}

	if err := rt.launcher.Render(items, w); err != nil {
		return fmt.Errorf("rendering search results: %w", err)
	}
	return nil
}
