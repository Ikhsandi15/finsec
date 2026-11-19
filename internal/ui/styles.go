package ui

import (
	"github.com/charmbracelet/lipgloss"

	"finsec-term/internal/model"
)

// Bloomberg-inspired Color Palette
var (
	ColorAmber       = lipgloss.Color("#FFB000") // Classic Bloomberg Amber
	ColorAmberDark   = lipgloss.Color("#D49000")
	ColorAmberDim    = lipgloss.Color("#7F5A00")
	ColorBlack       = lipgloss.Color("#0C0C0C")
	ColorDarkBg      = lipgloss.Color("#141414")
	ColorPanelBg     = lipgloss.Color("#1A1A1A")
	ColorTextMuted   = lipgloss.Color("#707070")
	ColorTextLight   = lipgloss.Color("#D8D8D8")
	ColorTextBright  = lipgloss.Color("#FFFFFF")
	ColorBorderDim   = lipgloss.Color("#2E2E2E")
	ColorRedCrit     = lipgloss.Color("#FF3333") // Stack Crit / Vulnerability
	ColorMagentaPol  = lipgloss.Color("#D63384") // ID Politics
	ColorCyanGlobal  = lipgloss.Color("#00D2D3") // Global Macro
	ColorGreenSecure = lipgloss.Color("#10B981")
	ColorBlueInfo    = lipgloss.Color("#3B82F6")
)

// Lipgloss Styles
var (
	// Header
	StyleHeaderBar = lipgloss.NewStyle().
			Bold(true).
			Background(ColorDarkBg).
			Foreground(ColorAmber).
			Padding(0, 1)

	StyleHeaderTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAmber).
				PaddingRight(2)

	StyleHeaderClock = lipgloss.NewStyle().
				Foreground(ColorCyanGlobal).
				Bold(true)

	StyleHeaderUnread = lipgloss.NewStyle().
				Foreground(ColorRedCrit).
				Bold(true)

	StyleHeaderFilter = lipgloss.NewStyle().
				Foreground(ColorGreenSecure).
				Bold(true)

	// Panes & Borders
	StylePaneInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderDim)

	StylePaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAmber).
			Bold(true)

	StylePaneTitleInactive = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextMuted)

	StylePaneTitleActive = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAmber)

	// List Items
	StyleItemNormal = lipgloss.NewStyle().
			Foreground(ColorTextLight)

	StyleItemRead = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleItemSelected = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlack).
				Background(ColorAmber)

	StyleItemTime = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	StyleItemSource = lipgloss.NewStyle().
			Foreground(ColorCyanGlobal)

	// Badges
	StyleBadgeStackCrit = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextBright).
				Background(ColorRedCrit).
				Padding(0, 1)

	StyleBadgeIDMacro = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlack).
				Background(ColorAmber).
				Padding(0, 1)

	StyleBadgeIDPol = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextBright).
			Background(ColorMagentaPol).
			Padding(0, 1)

	StyleBadgeGlobalMkt = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlack).
				Background(ColorCyanGlobal).
				Padding(0, 1)

	StyleBadgeCVE = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextBright).
			Background(ColorRedCrit).
			Padding(0, 1)

	// Inspector Pane
	StyleInspectorTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAmber)

	StyleInspectorMetaKey = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextMuted)

	StyleInspectorMetaVal = lipgloss.NewStyle().
				Foreground(ColorTextLight)

	StyleInspectorDesc = lipgloss.NewStyle().
				Foreground(ColorTextLight)

	StyleInspectorURL = lipgloss.NewStyle().
				Foreground(ColorCyanGlobal).
				Underline(true)

	// Bottom Command Bar
	StyleStatusBar = lipgloss.NewStyle().
			Background(ColorDarkBg).
			Foreground(ColorTextMuted).
			Padding(0, 1)

	StyleStatusKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAmber)

	StyleStatusDesc = lipgloss.NewStyle().
			Foreground(ColorTextLight)

	StyleFilterPrompt = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorGreenSecure)
)

// RenderBadge renders the appropriate lipgloss styled badge
func RenderBadge(badge string) string {
	switch badge {
	case model.BadgeStackCrit:
		return StyleBadgeStackCrit.Render("[STACK-CRIT]")
	case model.BadgeIDMacro:
		return StyleBadgeIDMacro.Render("[ID-MACRO]")
	case model.BadgeIDPol:
		return StyleBadgeIDPol.Render("[ID-POL]")
	case model.BadgeGlobalMkt:
		return StyleBadgeGlobalMkt.Render("[GLOBAL-MKT]")
	case model.BadgeCVE:
		return StyleBadgeCVE.Render("[CVE]")
	default:
		return lipgloss.NewStyle().Foreground(ColorTextMuted).Render("[" + badge + "]")
	}
}
