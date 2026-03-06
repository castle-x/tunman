package ui

import (
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func trimToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	result := make([]rune, 0, len(runes))
	for _, r := range runes {
		candidate := string(append(result, r))
		if lipgloss.Width(candidate) >= width {
			break
		}
		result = append(result, r)
	}
	if len(result) == 0 {
		return ""
	}
	if lipgloss.Width(string(result)) >= width {
		return string(result)
	}
	return string(result) + "…"
}

func fitLines(lines []string, height int, width int) []string {
	if height <= 0 {
		return []string{}
	}
	out := make([]string, 0, height)
	for _, line := range lines {
		out = append(out, trimToWidth(line, max(1, width)))
		if len(out) >= height {
			return out[:height]
		}
	}
	for len(out) < height {
		out = append(out, "")
	}
	return out
}

var fullwidthRange = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0xFF01, Hi: 0xFF60, Stride: 1},
		{Lo: 0xFFE0, Hi: 0xFFE6, Stride: 1},
	},
}

func isCJK(r rune) bool {
	return unicode.In(r,
		unicode.Han,
		unicode.Katakana,
		unicode.Hiragana,
		unicode.Hangul,
		fullwidthRange,
	)
}

func termWidth(s string) int {
	w := 0
	for _, r := range s {
		if isCJK(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

func padRight(s string, targetWidth int) string {
	w := termWidth(s)
	for w < targetWidth {
		s += " "
		w++
	}
	return s
}

func emptyFallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func notifyCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return notifyMsg(message)
	}
}

func refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshMsg{}
	}
}
