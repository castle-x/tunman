package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/tunman/internal/core"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/model"
)

// LogsModel 日志页面
type LogsModel struct {
	tunnel *model.Tunnel
	lines  []string
	offset int
	follow bool
	viewH  int
}

func NewLogsModel() LogsModel {
	return LogsModel{follow: true}
}

func (m *LogsModel) SetTunnel(tunnel *model.Tunnel) {
	m.tunnel = tunnel
	m.lines = nil
	m.offset = 0
	m.follow = true
	m.viewH = 0
}

func (m LogsModel) Update(msg tea.Msg, controller *core.Controller, _ int, height int) (LogsModel, tea.Cmd) {
	if m.tunnel == nil {
		return m, nil
	}

	pageSize := max(DefaultPageSize, height/2)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			m.follow = false
			m.offset--
		case "down", "j":
			m.follow = false
			m.offset++
		case "pgup":
			m.follow = false
			m.offset -= pageSize
		case "pgdown":
			m.follow = false
			m.offset += pageSize
		case "home", "g":
			m.follow = false
			m.offset = 0
		case "end", "G":
			m.follow = true
		case "t":
			m.follow = !m.follow
		case "r":
			// keep current offset/follow mode and refresh logs below
		}
	}

	raw, err := controller.GetLogs(m.tunnel, false)
	if err != nil {
		m.lines = []string{i18n.Tf("logs_load_failed", err.Error())}
	} else {
		m.lines = normalizeLogs(raw)
	}

	m.viewH = max(1, height-5)
	maxOffset := max(0, len(m.lines)-m.viewH)
	if m.follow {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}

	return m, nil
}

func (m LogsModel) View(width, height int) string {
	lines := make([]string, 0, height)

	if m.tunnel == nil {
		lines = append(lines, styleHint.Render(i18n.T("logs_no_tunnel")))
		return strings.Join(fitLines(lines, height, width), "\n")
	}

	mode := i18n.T("logs_follow_on")
	if !m.follow {
		mode = i18n.T("logs_follow_off")
	}
	head := i18n.Tf("logs_head", m.tunnel.Name, m.tunnel.Category.DisplayName(), m.tunnel.Status, mode, len(m.lines))
	lines = append(lines, styleHint.Render(trimToWidth(head, width)))
	lines = append(lines, sectionDivider(width))

	bodyHeight := max(1, height-len(lines)-1)
	if len(m.lines) == 0 {
		lines = append(lines, styleHint.Render(i18n.T("logs_empty")))
	} else {
		end := min(len(m.lines), m.offset+bodyHeight)
		for i := m.offset; i < end; i++ {
			row := fmt.Sprintf("%5d  %s", i+1, m.lines[i])
			lines = append(lines, styleNormalRow.Render(trimToWidth(row, width)))
		}
	}

	shownFrom := 0
	shownTo := 0
	if len(m.lines) > 0 {
		shownFrom = m.offset + 1
		shownTo = min(len(m.lines), m.offset+bodyHeight)
	}
	footer := i18n.Tf("logs_window", shownFrom, shownTo, len(m.lines))
	lines = append(lines, styleHint.Render(trimToWidth(footer, width)))

	return strings.Join(fitLines(lines, height, width), "\n")
}

func (m LogsModel) ContextLine() string {
	if m.tunnel == nil {
		return i18n.T("logs_context_none")
	}

	mode := i18n.T("logs_follow_on")
	if !m.follow {
		mode = i18n.T("logs_follow_off")
	}

	from := 0
	to := 0
	if len(m.lines) > 0 {
		from = m.offset + 1
		to = min(len(m.lines), m.offset+m.viewH)
	}
	return i18n.Tf("logs_context", m.tunnel.Name, m.tunnel.Status, mode, from, to, len(m.lines))
}

func normalizeLogs(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimRight(raw, "\n")
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, "\n")
}
