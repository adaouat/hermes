package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newConfigurationCmd(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "configuration",
		Short: "Print the merged per-product configuration as JSON",
		RunE: func(c *cobra.Command, _ []string) error {
			return runConfiguration(c.OutOrStdout(), rt)
		},
	}
}

func runConfiguration(w io.Writer, rt *runtime) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rt.config); err != nil {
		return fmt.Errorf("encoding configuration: %w", err)
	}
	return nil
}
