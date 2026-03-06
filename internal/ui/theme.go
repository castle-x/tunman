package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	SeparatorWidth  = 60
	DefaultPageSize = 10
)

var (
	colorBgMain       = lipgloss.Color("#0F1117")
	colorBgHeader     = lipgloss.Color("#131722")
	colorBgInfo       = lipgloss.Color("#111A1E")
	colorBgFooter     = lipgloss.Color("#131722")
	colorBgTableHead  = lipgloss.Color("#1B1F2B")
	colorBgSelected   = lipgloss.Color("#17322E")
	colorBgSearch     = lipgloss.Color("#333333")
	colorPrimary      = lipgloss.Color("#00D4AA")
	colorSecondary    = lipgloss.Color("#FF6B6B")
	colorAccent       = lipgloss.Color("#4ECDC4")
	colorDim          = lipgloss.Color("#666666")
	colorWhite        = lipgloss.Color("#FFFFFF")
	colorCyan         = lipgloss.Color("#5A9FB8")
	colorTextNormal   = lipgloss.Color("#AAAAAA")
	colorSuccess      = lipgloss.Color("#00D4AA")
	colorDanger       = lipgloss.Color("#FF6B6B")
	colorWarn         = lipgloss.Color("#FFCC66")
	colorInfoLabel    = lipgloss.Color("#7BC6CF")
	colorInfoValue    = lipgloss.Color("#D8DFE9")
	colorContentTitle = lipgloss.Color("#FFFFFF")
)

var (
	styleHeaderBlock = lipgloss.NewStyle().
				Foreground(colorWhite).
				Padding(0, 1)

	styleInfoBar = lipgloss.NewStyle().
			Foreground(colorInfoValue).
			Padding(0, 1)

	styleContentBlock = lipgloss.NewStyle().
				Foreground(colorTextNormal).
				Padding(0, 1)

	styleFooterBlock = lipgloss.NewStyle().
				Foreground(colorTextNormal).
				Padding(0, 1)

	styleLogo = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	styleAccent = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	stylePageTitle = lipgloss.NewStyle().
			Foreground(colorContentTitle).
			Bold(true)

	styleSelectedRow = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	styleNormalRow = lipgloss.NewStyle().
			Foreground(colorTextNormal)

	styleSelectable = lipgloss.NewStyle().
			Foreground(colorCyan)

	styleCursor = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleHint = lipgloss.NewStyle().
			Foreground(colorDim)

	styleWarn = lipgloss.NewStyle().
			Foreground(colorWarn)

	styleError = lipgloss.NewStyle().
			Foreground(colorDanger)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess)

	styleInfoLabel = lipgloss.NewStyle().
			Foreground(colorInfoLabel).
			Bold(true)

	styleInfoValue = lipgloss.NewStyle().
			Foreground(colorInfoValue)

	styleHeaderCell = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	styleSearchInput = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(colorBgSearch)

	styleActiveTab = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 1)

	styleInactiveTab = lipgloss.NewStyle().
				Foreground(colorDim).
				Padding(0, 1)

	styleStatBadge = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
)

func sectionDivider(widths ...int) string {
	w := SeparatorWidth
	if len(widths) > 0 && widths[0] > 0 {
		w = widths[0]
	}
	return lipgloss.NewStyle().
		Foreground(colorDim).
		Render(strings.Repeat("─", w))
}

func infoLabel(key, value string) string {
	return styleInfoLabel.Render(key) + styleInfoValue.Render(value)
}

func statusMark(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return styleSuccess.Render("✓")
	case "error":
		return styleError.Render("✗")
	default:
		return styleHint.Render("·")
	}
}
