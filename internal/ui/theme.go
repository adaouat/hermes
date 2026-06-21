package ui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	forgeui "github.com/adaouat/forge/ui"
)

// Accent is hermes's brand — Mercurial: steel/sky blue title/program/flags, slate/silver
// commands — over forge's shared palette (forge ADR-0010).
func Accent() forgeui.Accent {
	return forgeui.Accent{
		Light:          lipgloss.Color("#4A6FA5"),
		Dark:           lipgloss.Color("#93C5FD"),
		SecondaryLight: lipgloss.Color("#475569"),
		SecondaryDark:  lipgloss.Color("#CBD5E1"),
	}
}

// HuhTheme is hermes's branded interactive-form theme.
func HuhTheme() huh.ThemeFunc { return forgeui.HuhTheme(Accent()) }
