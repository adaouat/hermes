package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/adaouat/hermes/internal/jetbrains"
)

func newAllCmd(rt *runtime) *cobra.Command {
	var filter string

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Search recent projects across every installed JetBrains product",
		RunE: func(c *cobra.Command, _ []string) error {
			return runAll(c.OutOrStdout(), rt, filter)
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "", "Filter projects by name or path")
	return cmd
}

func runAll(w io.Writer, rt *runtime, filter string) error {
	svc := jetbrains.NewService(rt.fs, rt.env, rt.config)
	items := svc.SearchAll(filter)

	if err := rt.launcher.Render(items, w); err != nil {
		return fmt.Errorf("rendering search results: %w", err)
	}
	return nil
}
