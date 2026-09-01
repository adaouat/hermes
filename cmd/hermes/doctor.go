package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adaouat/forge/ui"
	"github.com/adaouat/hermes/internal/jetbrains"
	"github.com/adaouat/hermes/pkg/domain"
)

func newDoctorCmd(rt *runtime) *cobra.Command {
	var productFlag string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Explain why a product's recent projects can or can't be located",
		RunE: func(c *cobra.Command, _ []string) error {
			return runDoctor(c.OutOrStdout(), rt, productFlag)
		},
	}
	cmd.Flags().StringVar(&productFlag, "product", "", `Limit to one JetBrains product (default: every supported product)`)
	return cmd
}

func runDoctor(w io.Writer, rt *runtime, productFlag string) error {
	products, err := doctorTargets(productFlag)
	if err != nil {
		return err
	}

	for i, product := range products {
		writeDoctorReport(w, rt, i+1, len(products), product)
	}
	return nil
}

func doctorTargets(productFlag string) ([]domain.Product, error) {
	if productFlag == "" {
		return domain.Products(), nil
	}
	p, err := parseProduct(productFlag)
	if err != nil {
		return nil, err
	}
	return []domain.Product{p}, nil
}

// writeDoctorReport prints one product's diagnosis: paths searched/matched (via
// Locator/Repository's existing *NotFoundError, which already carries SearchedPaths and
// Names), the settings-directory regex applied, and which recents file (if any) would be
// read. Stops at the first missing piece - a later step can't run without an earlier one
// (e.g. no settings directory means no recents file to report).
func writeDoctorReport(w io.Writer, rt *runtime, n, total int, product domain.Product) {
	label := fmt.Sprintf("[%d/%d] %s", n, total, product.Display())

	details, ok := rt.config[product]
	if !ok {
		printDoctorStatus(w, false, label, "no configuration for this product")
		return
	}

	locator := jetbrains.NewLocator(rt.fs, rt.env, product, details)
	repo := jetbrains.NewRepository(rt.fs, rt.env, product, details)

	appPath, err := locator.LocateApplication()
	if err != nil {
		printDoctorStatus(w, false, label, err.Error())
		return
	}
	binPath, err := locator.LocateBin()
	if err != nil {
		printDoctorStatus(w, false, label, err.Error())
		return
	}

	printDoctorStatus(w, true, label, "found")
	printDoctorDetail(w, "application: "+appPath)
	printDoctorDetail(w, "binary: "+binPath)
	printDoctorDetail(w, "settings regex: "+jetbrains.SettingsRegexp(product, details.PreferencePrefix).String())

	settingsDir, err := repo.LocateSettingsDirectory()
	if err != nil {
		printDoctorDetail(w, err.Error())
		return
	}
	printDoctorDetail(w, "settings directory: "+settingsDir)

	recents, err := repo.RecentsFiles()
	if err != nil {
		printDoctorDetail(w, err.Error())
		return
	}
	if len(recents) == 0 {
		printDoctorDetail(w, "recents file: none found (no projects opened yet)")
		return
	}
	printDoctorDetail(w, "recents file: "+strings.Join(recents, ", "))
}

// printDoctorStatus renders label's outcome via ui.Success/ui.Warn directly rather than
// Spinner.Step - Step always renders a green checkmark, which would misrepresent a
// not-found product as a success.
func printDoctorStatus(w io.Writer, found bool, label, detail string) {
	line := label + " — " + detail
	if found {
		_, _ = fmt.Fprintln(w, ui.Success(w, line))
		return
	}
	_, _ = fmt.Fprintln(w, ui.Warn(w, line))
}

func printDoctorDetail(w io.Writer, msg string) {
	_, _ = fmt.Fprintln(w, ui.Info(w, msg))
}
