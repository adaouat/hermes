package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	forgeexit "github.com/adaouat/forge/exitcode"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/pkg/domain"
)

func newOpenCmd(rt *runtime) *cobra.Command {
	var product, path string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Resolve one project path into a single launcher item",
		RunE: func(c *cobra.Command, _ []string) error {
			return runOpen(c.OutOrStdout(), rt, product, path)
		},
	}
	cmd.Flags().StringVar(&product, "product", "", `JetBrains product (see "hermes configuration" for the full list)`)
	cmd.Flags().StringVar(&path, "path", "", "Path to the project")
	_ = cmd.MarkFlagRequired("product")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

func runOpen(w io.Writer, rt *runtime, productFlag, path string) error {
	product, err := parseProduct(productFlag)
	if err != nil {
		return err
	}

	svc := jetbrains.NewService(rt.fs, rt.env, rt.config)
	item, err := svc.Open(product, path)
	if err != nil {
		var notFound *jetbrains.NotFoundError
		if errors.As(err, &notFound) {
			return forgeexit.Wrap(exitNotFound, err)
		}
		return err
	}

	if err := rt.launcher.Render([]domain.Item{item}, w); err != nil {
		return fmt.Errorf("rendering opened project: %w", err)
	}
	return nil
}
