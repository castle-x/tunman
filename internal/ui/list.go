package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/tunman/internal/core"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/model"
)

// ListModel 列表页面模型
type ListModel struct {
	cursor         int
	offset         int
	filter         model.Category
	searchInput    textinput.Model
	searching      bool
	filteredCached []model.Tunnel
	openLogs       bool
	openEdit       bool
	openCreate     bool
	openDelete     bool
}

// NewListModel 创建列表模型
func NewListModel() ListModel {
	search := textinput.New()
	search.Placeholder = i18n.T("list_search_placeholder")
	search.Prompt = i18n.T("list_search_prompt")
	search.CharLimit = 80
	search.Width = 48

	return ListModel{
		filter:      "",
		searchInput: search,
	}
}

// Init 初始化
func (m ListModel) Init() tea.Cmd {
	return nil
}

// Update 更新
func (m ListModel) Update(
	msg tea.Msg,
	tunnels []model.Tunnel,
	_ *core.Storage,
	controller *core.Controller,
	_ int,
	height int,
) (ListModel, tea.Cmd) {
	m.openLogs = false
	m.openEdit = false
	m.openCreate = false
	m.openDelete = false
	m.filteredCached = m.filterTunnels(tunnels)
	m.clampCursor()
	pageSize := max(DefaultPageSize, height-8)
	m.ensureOffset(pageSize)

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.searching {
		switch keyMsg.String() {
		case "enter", "esc":
			m.searching = false
			m.searchInput.Blur()
			m.filteredCached = m.filterTunnels(tunnels)
			m.clampCursor()
			m.ensureOffset(pageSize)
			return m, nil
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(keyMsg)
		m.filteredCached = m.filterTunnels(tunnels)
		m.clampCursor()
		m.ensureOffset(pageSize)
		return m, cmd
	}

	switch keyMsg.String() {
	case "/":
		m.searching = true
		m.searchInput.Focus()
		return m, nil

	case "f":
		m.filter = nextCategoryFilter(m.filter)
		m.filteredCached = m.filterTunnels(tunnels)
		m.clampCursor()
		m.ensureOffset(pageSize)
		return m, notifyCmd(i18n.Tf("list_filter_switched", filterLabel(m.filter)))

	case "up", "k":
		m.moveCursor(-1, pageSize, true)
		return m, nil
	case "down", "j":
		m.moveCursor(1, pageSize, true)
		return m, nil
	case "pgup":
		m.moveCursor(-pageSize, pageSize, false)
		return m, nil
	case "pgdown":
		m.moveCursor(pageSize, pageSize, false)
		return m, nil
	case "home", "g":
		m.cursor = 0
		m.ensureOffset(pageSize)
		return m, nil
	case "end", "G":
		if len(m.filteredCached) > 0 {
			m.cursor = len(m.filteredCached) - 1
		}
		m.ensureOffset(pageSize)
		return m, nil

	case "enter", "l":
		if m.Selected() == nil {
			return m, notifyCmd(i18n.T("err_select_tunnel_first"))
		}
		m.openLogs = true
		return m, nil

	case "s":
		return m, m.runAction("start", tunnels, controller)
	case "x":
		return m, m.runAction("stop", tunnels, controller)
	case "r":
		return m, m.runAction("restart", tunnels, controller)
	case "S":
		return m, m.runAction("start", tunnels, controller)
	case "X":
		return m, m.runAction("stop", tunnels, controller)
	case "R":
		return m, m.runAction("restart", tunnels, controller)

	case "a":
		m.openCreate = true
		return m, nil

	case "d":
		m.openDelete = true
		return m, nil

	case "e":
		if m.Selected() == nil {
			return m, notifyCmd(i18n.T("err_select_tunnel_first"))
		}
		m.openEdit = true
		return m, nil

	case "y":
		selected := m.Selected()
		if selected == nil {
			return m, notifyCmd(i18n.T("err_select_tunnel_first"))
		}
		if err := clipboard.WriteAll(selected.DisplayURL()); err != nil {
			return m, notifyCmd(i18n.Tf("err_copy_failed", err.Error()))
		}
		return m, notifyCmd(i18n.Tf("list_copied_url", selected.DisplayURL()))
	}

	return m, nil
}

// View 渲染
func (m ListModel) View(tunnels []model.Tunnel, width, height int) string {
	filtered, cursor, offset := m.snapshot(tunnels, height)

	nameW := 22
	if width >= 130 {
		nameW = 30
	} else if width < 90 {
		nameW = 16
	}
	endpointW := max(14, width-66)

	lines := make([]string, 0, height)

	if m.searching {
		lines = append(lines, styleSearchInput.Render(trimToWidth(m.searchInput.View(), width)))
	}

	header := fmt.Sprintf("  %s  %s  %s  %s  %s",
		padRight("", 1),
		padRight(i18n.T("list_col_category"), 11),
		padRight(i18n.T("list_col_name"), nameW),
		padRight(i18n.T("list_col_port"), 6),
		i18n.T("list_col_endpoint"),
	)
	lines = append(lines, styleHeaderCell.Render(trimToWidth(header, width)))

	bodyHeight := max(1, height-len(lines)-2)
	if len(filtered) == 0 {
		lines = append(lines, styleHint.Render(i18n.T("list_empty")))
	} else {
		end := min(len(filtered), offset+bodyHeight)
		for i := offset; i < end; i++ {
			tunnel := filtered[i]
			prefix := "  "
			if i == cursor {
				prefix = styleCursor.Render("❯") + " "
			}

			catText := padRight(tunnel.Category.DisplayName(), 11)
			nameText := padRight(tunnel.Name, nameW)
			portText := fmt.Sprintf("%-6d", tunnel.Port)
			endText := trimToWidth(tunnel.DisplayURL(), endpointW)

			style := styleNormalRow
			if i == cursor {
				style = styleSelectedRow
			}
			row := fmt.Sprintf("%s%s  %s  %s  %s  %s",
				prefix,
				statusMark(string(tunnel.Status)),
				style.Render(catText),
				style.Render(nameText),
				style.Render(portText),
				style.Render(endText),
			)
			lines = append(lines, trimToWidth(row, width))
		}
	}

	return strings.Join(fitLines(lines, height, width), "\n")
}

func (m ListModel) ContextLine(tunnels []model.Tunnel) string {
	filtered, cursor, _ := m.snapshot(tunnels, 20)
	if selected := m.selectedFrom(filtered, cursor); selected != nil {
		return fmt.Sprintf("%s  %s  %s", selected.Name, selected.Category.DisplayName(), selected.Status)
	}
	if len(filtered) == 0 {
		return i18n.T("list_empty_short")
	}
	return fmt.Sprintf("%d %s", len(filtered), i18n.T("list_count_suffix"))
}

func (m ListModel) Selected() *model.Tunnel {
	return m.selectedFrom(m.filteredCached, m.cursor)
}

func (m *ListModel) ConsumeOpenLogs() bool {
	if !m.openLogs {
		return false
	}
	m.openLogs = false
	return true
}

func (m ListModel) snapshot(tunnels []model.Tunnel, height int) ([]model.Tunnel, int, int) {
	filtered := m.filterTunnels(tunnels)
	cursor := m.cursor
	if len(filtered) == 0 {
		return filtered, 0, 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(filtered) {
		cursor = len(filtered) - 1
	}

	bodyHeight := max(1, height-7)
	offset := m.offset
	maxOffset := max(0, len(filtered)-bodyHeight)
	if offset > maxOffset {
		offset = maxOffset
	}
	if cursor < offset {
		offset = cursor
	}
	if cursor >= offset+bodyHeight {
		offset = cursor - bodyHeight + 1
	}

	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	return filtered, cursor, offset
}

func (m ListModel) selectedFrom(tunnels []model.Tunnel, cursor int) *model.Tunnel {
	if len(tunnels) == 0 || cursor < 0 || cursor >= len(tunnels) {
		return nil
	}
	tunnel := tunnels[cursor]
	return &tunnel
}

func (m *ListModel) clampCursor() {
	if len(m.filteredCached) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filteredCached) {
		m.cursor = len(m.filteredCached) - 1
	}
}

func (m *ListModel) ensureOffset(pageSize int) {
	if len(m.filteredCached) == 0 {
		m.offset = 0
		return
	}
	maxOffset := max(0, len(m.filteredCached)-pageSize)
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

func (m *ListModel) moveCursor(delta, pageSize int, wrap bool) {
	if len(m.filteredCached) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}

	next := m.cursor + delta
	if wrap {
		for next < 0 {
			next += len(m.filteredCached)
		}
		next = next % len(m.filteredCached)
	} else {
		if next < 0 {
			next = 0
		}
		if next >= len(m.filteredCached) {
			next = len(m.filteredCached) - 1
		}
	}
	m.cursor = next
	m.ensureOffset(pageSize)
}

func (m ListModel) filterTunnels(tunnels []model.Tunnel) []model.Tunnel {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	filtered := make([]model.Tunnel, 0, len(tunnels))

	for _, tunnel := range tunnels {
		if m.filter != "" && tunnel.Category != m.filter {
			continue
		}
		if query == "" {
			filtered = append(filtered, tunnel)
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			tunnel.ID,
			tunnel.Name,
			tunnel.BaseDomain,
			tunnel.Prefix,
			tunnel.FullDomain(),
			tunnel.DisplayURL(),
			string(tunnel.Category),
		}, " "))
		if strings.Contains(haystack, query) {
			filtered = append(filtered, tunnel)
		}
	}

	return filtered
}

func (m *ListModel) ConsumeOpenEdit() bool {
	if !m.openEdit {
		return false
	}
	m.openEdit = false
	return true
}

func (m *ListModel) ConsumeOpenCreate() bool {
	if !m.openCreate {
		return false
	}
	m.openCreate = false
	return true
}

func (m *ListModel) ConsumeOpenDelete() bool {
	if !m.openDelete {
		return false
	}
	m.openDelete = false
	return true
}

func (m ListModel) runAction(action string, tunnels []model.Tunnel, controller *core.Controller) tea.Cmd {
	selected := m.Selected()
	if selected == nil {
		return notifyCmd(i18n.T("err_select_tunnel_first"))
	}

	var ptr *model.Tunnel
	for i := range tunnels {
		if tunnels[i].ID == selected.ID {
			ptr = &tunnels[i]
			break
		}
	}
	if ptr == nil {
		return notifyCmd(i18n.T("err_select_tunnel_first"))
	}

	var err error
	switch action {
	case "start":
		if ptr.Status == model.StatusRunning {
			return notifyCmd(i18n.Tf("list_action_skip", actionName(action), selected.Name))
		}
		err = controller.Start(ptr)
	case "stop":
		if ptr.Status != model.StatusRunning {
			return notifyCmd(i18n.Tf("list_action_skip", actionName(action), selected.Name))
		}
		err = controller.Stop(ptr)
	case "restart":
		err = controller.Restart(ptr)
	}
	if err != nil {
		return tea.Batch(refreshCmd(), notifyCmd(i18n.Tf("list_action_err", actionName(action), err)))
	}
	return tea.Batch(refreshCmd(), notifyCmd(i18n.Tf("list_action_ok", actionName(action), selected.Name)))
}

func nextCategoryFilter(current model.Category) model.Category {
	switch current {
	case "":
		return model.CategoryCustom
	case model.CategoryCustom:
		return model.CategoryTesting
	case model.CategoryTesting:
		return model.CategoryEphemeral
	default:
		return ""
	}
}

func filterLabel(category model.Category) string {
	switch category {
	case model.CategoryCustom:
		return i18n.T("cat_custom")
	case model.CategoryTesting:
		return i18n.T("cat_testing")
	case model.CategoryEphemeral:
		return i18n.T("cat_ephemeral")
	default:
		return i18n.T("cat_all")
	}
}

func searchLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return i18n.T("meta_none")
	}
	return value
}

func actionName(action string) string {
	switch action {
	case "start":
		return i18n.T("action_start")
	case "stop":
		return i18n.T("action_stop")
	case "restart":
		return i18n.T("action_restart")
	default:
		return action
	}
}
