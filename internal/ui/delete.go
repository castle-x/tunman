package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/tunman/internal/core"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/model"
)

// DeleteModel 删除分支模型
type DeleteModel struct {
	cursor    int
	offset    int
	confirm   bool
	exit      bool
	pendingID string
}

func NewDeleteModel() DeleteModel {
	return DeleteModel{}
}

func (m DeleteModel) Update(
	msg tea.Msg,
	tunnels []model.Tunnel,
	storage *core.Storage,
	controller *core.Controller,
	_ int,
	height int,
) (DeleteModel, tea.Cmd) {
	pageSize := max(DefaultPageSize, height-7)
	m.clampCursor(len(tunnels))
	m.ensureOffset(len(tunnels), pageSize)

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.confirm {
		switch keyMsg.String() {
		case "y":
			return m.deletePending(tunnels, storage, controller)
		case "n", "esc", "b":
			m.confirm = false
			m.pendingID = ""
			return m, notifyCmd(i18n.T("delete_canceled"))
		default:
			return m, nil
		}
	}

	switch keyMsg.String() {
	case "up", "k":
		m.cursor--
		m.clampCursor(len(tunnels))
		m.ensureOffset(len(tunnels), pageSize)
		return m, nil

	case "down", "j":
		m.cursor++
		m.clampCursor(len(tunnels))
		m.ensureOffset(len(tunnels), pageSize)
		return m, nil

	case "pgup":
		m.cursor -= pageSize
		m.clampCursor(len(tunnels))
		m.ensureOffset(len(tunnels), pageSize)
		return m, nil

	case "pgdown":
		m.cursor += pageSize
		m.clampCursor(len(tunnels))
		m.ensureOffset(len(tunnels), pageSize)
		return m, nil

	case "home", "g":
		m.cursor = 0
		m.clampCursor(len(tunnels))
		m.ensureOffset(len(tunnels), pageSize)
		return m, nil

	case "end", "G":
		m.cursor = len(tunnels) - 1
		m.clampCursor(len(tunnels))
		m.ensureOffset(len(tunnels), pageSize)
		return m, nil

	case "enter", " ":
		selected := m.selected(tunnels)
		if selected == nil {
			return m, notifyCmd(i18n.T("err_select_tunnel_first"))
		}
		m.confirm = true
		m.pendingID = selected.ID
		return m, notifyCmd(i18n.Tf("delete_confirm_prompt", selected.Name))

	case "esc", "b":
		m.exit = true
		return m, nil
	}

	return m, nil
}

func (m DeleteModel) View(tunnels []model.Tunnel, width, height int) string {
	lines := make([]string, 0, height)

	header := fmt.Sprintf("  %s  %s  %s  %s  %s",
		padRight("", 1),
		padRight(i18n.T("list_col_category"), 11),
		padRight(i18n.T("list_col_name"), 24),
		padRight(i18n.T("list_col_port"), 6),
		i18n.T("list_col_endpoint"),
	)
	lines = append(lines, styleHeaderCell.Render(trimToWidth(header, width)))

	bodyHeight := max(1, height-len(lines)-2)
	if len(tunnels) == 0 {
		lines = append(lines, styleHint.Render(i18n.T("delete_empty")))
	} else {
		start := m.offset
		end := min(len(tunnels), start+bodyHeight)
		for i := start; i < end; i++ {
			tunnel := tunnels[i]
			prefix := "  "
			if i == m.cursor {
				prefix = styleCursor.Render("❯") + " "
			}
			pendingMark := " "
			if m.pendingID != "" && tunnel.ID == m.pendingID {
				pendingMark = styleWarn.Render("!")
			}

			catText := padRight(tunnel.Category.DisplayName(), 11)
			nameText := padRight(tunnel.Name, 24)
			portText := fmt.Sprintf("%-6d", tunnel.Port)
			endText := trimToWidth(tunnel.DisplayURL(), max(8, width-54))

			style := styleNormalRow
			if i == m.cursor {
				style = styleSelectedRow
			}
			row := fmt.Sprintf("%s%s  %s  %s  %s  %s",
				prefix,
				pendingMark,
				style.Render(catText),
				style.Render(nameText),
				style.Render(portText),
				style.Render(endText),
			)
			lines = append(lines, trimToWidth(row, width))
		}
	}

	if m.confirm {
		lines = append(lines, styleWarn.Render(i18n.T("delete_confirm_line")))
	} else if selected := m.selected(tunnels); selected != nil {
		lines = append(lines, styleHint.Render(trimToWidth(i18n.Tf("delete_current", selected.Name, selected.Status), width)))
	} else {
		lines = append(lines, styleHint.Render(i18n.T("list_current_none")))
	}

	return strings.Join(fitLines(lines, height, width), "\n")
}

func (m DeleteModel) ContextLine(tunnels []model.Tunnel) string {
	if selected := m.selected(tunnels); selected != nil {
		if m.confirm {
			return i18n.Tf("delete_context_confirm", selected.Name)
		}
		return i18n.Tf("delete_context_target", selected.Name, selected.Status)
	}
	return i18n.T("delete_context_none")
}

func (m *DeleteModel) ConsumeExit() bool {
	if !m.exit {
		return false
	}
	m.exit = false
	return true
}

func (m DeleteModel) deletePending(tunnels []model.Tunnel, storage *core.Storage, controller *core.Controller) (DeleteModel, tea.Cmd) {
	selected := m.selectedByID(tunnels, m.pendingID)
	m.confirm = false
	m.pendingID = ""

	if selected == nil {
		return m, notifyCmd(i18n.T("delete_target_not_found"))
	}

	if selected.Status == model.StatusRunning {
		_ = controller.Stop(selected)
	}
	_ = controller.TeardownTunnel(selected)
	if err := storage.DeleteTunnel(selected.ID); err != nil {
		return m, notifyCmd(i18n.Tf("delete_failed", err.Error()))
	}

	if m.cursor > 0 {
		m.cursor--
	}

	return m, tea.Batch(refreshCmd(), notifyCmd(i18n.Tf("delete_done", selected.Name)))
}

func (m *DeleteModel) clampCursor(total int) {
	if total == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= total {
		m.cursor = total - 1
	}
}

func (m *DeleteModel) ensureOffset(total, pageSize int) {
	if total <= 0 {
		m.offset = 0
		return
	}
	maxOffset := max(0, total-pageSize)
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+pageSize {
		m.offset = m.cursor - pageSize + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
}

func (m DeleteModel) selected(tunnels []model.Tunnel) *model.Tunnel {
	if len(tunnels) == 0 {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(tunnels) {
		return nil
	}
	t := tunnels[m.cursor]
	return &t
}

func (m DeleteModel) selectedByID(tunnels []model.Tunnel, id string) *model.Tunnel {
	for i := range tunnels {
		if tunnels[i].ID == id {
			return &tunnels[i]
		}
	}
	return nil
}
