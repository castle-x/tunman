package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/tunman/internal/core"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/model"
)

// Page 当前页面类型
type Page int

const (
	PageHome Page = iota
	PageList
	PageCreate
	PageDelete
	PageLogs
	PageHelp
)

const (
	minContentHeight = 8
	maxContentHeight = 16
)

const logoWide = `
████████╗██╗   ██╗███╗   ██╗███╗   ███╗ █████╗ ███╗   ██╗
╚══██╔══╝██║   ██║████╗  ██║████╗ ████║██╔══██╗████╗  ██║
   ██║   ██║   ██║██╔██╗ ██║██╔████╔██║███████║██╔██╗ ██║
   ██║   ██║   ██║██║╚██╗██║██║╚██╔╝██║██╔══██║██║╚██╗██║
   ██║   ╚██████╔╝██║ ╚████║██║ ╚═╝ ██║██║  ██║██║ ╚████║
   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝
`

const logoNarrow = `
████████╗██╗   ██╗███╗   ██╗███╗   ███╗
╚══██╔══╝██║   ██║████╗  ██║████╗ ████║
   ██║   ██║   ██║██╔██╗ ██║██╔████╔██║
   ██║   ██║   ██║██║╚██╗██║██║╚██╔╝██║
   ██║   ╚██████╔╝██║ ╚████║██║ ╚═╝ ██║
   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝╚═╝     ╚═╝
`

// Model 主应用模型
type Model struct {
	storage    *core.Storage
	controller *core.Controller
	config     *model.Config
	workdir    string

	page    Page
	tunnels []model.Tunnel
	width   int
	height  int

	cfInstalled bool
	cfLoggedIn  bool

	listModel   ListModel
	createModel CreateModel
	deleteModel DeleteModel
	logsModel   LogsModel
	helpModel   HelpModel

	message string
	showMsg bool
	msgAt   int64
}

// NewModel 创建主应用模型
func NewModel() (*Model, error) {
	storage := core.NewStorage()
	config, err := storage.LoadConfig()
	if err != nil {
		config = model.DefaultConfig()
	}

	controller := core.NewController(storage)
	tunnels, _ := storage.LoadTunnels()
	tunnels = controller.SyncStatus(tunnels)

	m := &Model{
		storage:     storage,
		controller:  controller,
		config:      config,
		page:        PageList,
		tunnels:     tunnels,
		workdir:     ".",
		cfInstalled: controller.CheckCloudflared() == nil,
		cfLoggedIn:  controller.CheckAuth() == nil,
		listModel:   NewListModel(),
		createModel: NewCreateModel(),
		deleteModel: NewDeleteModel(),
		logsModel:   NewLogsModel(),
		helpModel:   NewHelpModel(),
	}

	if wd, err := filepath.Abs("."); err == nil {
		m.workdir = wd
	}

	return m, nil
}

// Init 初始化
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.listModel.Init(), m.tickCmd())
}

type tickMsg struct{}
type refreshMsg struct{}
type notifyMsg string

type clearMsg struct {
	stamp int64
}

type tunnelCreatedMsg struct {
	Name string
}

type editFinishedMsg struct {
	tunnelID string
	tmpPath  string
	err      error
}

func (m *Model) tickCmd() tea.Cmd {
	seconds := max(1, m.config.RefreshInterval)
	return tea.Tick(time.Duration(seconds)*time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

// Update 更新
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, GlobalKeys.Quit) {
			m.controller.CleanupEphemeral()
			return m, tea.Quit
		}

		if key.Matches(msg, GlobalKeys.Refresh) {
			return m, tea.Batch(m.refresh(), m.notify(i18n.T("app_status_refreshed")))
		}

		if m.page == PageList && key.Matches(msg, GlobalKeys.Help) {
			m.page = PageHelp
			return m, nil
		}

		if key.Matches(msg, GlobalKeys.Back) {
			switch m.page {
			case PageCreate, PageDelete, PageLogs, PageHelp:
				m.page = PageList
				return m, nil
			}
		}

	case tickMsg:
		if m.config.AutoRefresh {
			return m, tea.Batch(m.refresh(), m.tickCmd())
		}
		return m, m.tickCmd()

	case refreshMsg:
		tunnels, _ := m.storage.LoadTunnels()
		m.tunnels = m.controller.SyncStatus(tunnels)
		return m, nil

	case notifyMsg:
		m.message = string(msg)
		m.showMsg = true
		m.msgAt = time.Now().UnixNano()
		stamp := m.msgAt
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearMsg{stamp: stamp}
		})

	case clearMsg:
		if m.msgAt == msg.stamp {
			m.showMsg = false
		}
		return m, nil

	case tunnelCreatedMsg:
		m.page = PageList
		return m, tea.Batch(
			m.refresh(),
			m.notify(i18n.Tf("app_tunnel_created", msg.Name)),
		)

	case editFinishedMsg:
		if msg.err != nil {
			return m, m.notify(i18n.Tf("list_edit_failed", msg.err))
		}
		data, err := os.ReadFile(msg.tmpPath)
		os.Remove(msg.tmpPath)
		if err != nil {
			return m, m.notify(i18n.Tf("list_edit_failed", err))
		}
		var updated model.Tunnel
		if err := json.Unmarshal(data, &updated); err != nil {
			return m, m.notify(i18n.Tf("list_edit_failed", err))
		}
		updated.ID = msg.tunnelID
		if err := m.storage.UpdateTunnel(updated); err != nil {
			return m, m.notify(i18n.Tf("list_edit_failed", err))
		}
		if updated.Category != model.CategoryEphemeral && updated.TunnelID != "" {
			if err := m.controller.WriteConfigYML(&updated); err != nil {
				return m, m.notify(i18n.Tf("list_edit_failed", err))
			}
		}
		return m, tea.Batch(m.refresh(), m.notify(i18n.Tf("list_edit_success", updated.Name)))
	}

	var cmd tea.Cmd
	switch m.page {
	case PageList:
		newList, listCmd := m.listModel.Update(msg, m.tunnels, m.storage, m.controller, m.width, m.contentHeight())
		m.listModel = newList
		cmd = listCmd

		if m.listModel.ConsumeOpenLogs() {
			if selected := m.listModel.Selected(); selected != nil {
				m.logsModel.SetTunnel(selected)
				m.page = PageLogs
			}
		}

		if m.listModel.ConsumeOpenEdit() {
			if selected := m.listModel.Selected(); selected != nil {
				tmpPath, err := m.prepareEditFile(selected)
				if err != nil {
					return m, m.notify(i18n.Tf("list_edit_failed", err))
				}
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = m.config.Editor
				}
				tunnelID := selected.ID
				c := exec.Command(editor, tmpPath)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return editFinishedMsg{tunnelID: tunnelID, tmpPath: tmpPath, err: err}
				})
			}
		}

		if m.listModel.ConsumeOpenCreate() {
			m.page = PageCreate
		}

		if m.listModel.ConsumeOpenDelete() {
			m.page = PageDelete
		}

	case PageCreate:
		newCreate, createCmd := m.createModel.Update(msg, m.storage, m.controller, m.width, m.contentHeight())
		m.createModel = newCreate
		cmd = createCmd

		if m.createModel.ConsumeExit() {
			m.page = PageList
		}

	case PageDelete:
		newDelete, deleteCmd := m.deleteModel.Update(msg, m.tunnels, m.storage, m.controller, m.width, m.contentHeight())
		m.deleteModel = newDelete
		cmd = deleteCmd

		if m.deleteModel.ConsumeExit() {
			m.page = PageList
		}

	case PageLogs:
		newLogs, logsCmd := m.logsModel.Update(msg, m.controller, m.width, m.contentHeight())
		m.logsModel = newLogs
		cmd = logsCmd

	case PageHelp:
		newHelp, helpCmd := m.helpModel.Update(msg, m.width, m.contentHeight())
		m.helpModel = newHelp
		cmd = helpCmd
	}

	return m, cmd
}

// View 渲染
func (m *Model) View() string {
	if m.width == 0 {
		m.width = 100
	}
	if m.height == 0 {
		m.height = 28
	}

	header := m.renderHeader()
	info := m.renderInfoBar()
	footer := m.renderFooter()
	contentWidth := max(40, m.width-2)
	contentHeight := m.contentHeight()

	var contentRaw string
	switch m.page {
	case PageList:
		contentRaw = m.listModel.View(m.tunnels, contentWidth, contentHeight)
	case PageCreate:
		contentRaw = m.createModel.View(contentWidth, contentHeight)
	case PageDelete:
		contentRaw = m.deleteModel.View(m.tunnels, contentWidth, contentHeight)
	case PageLogs:
		contentRaw = m.logsModel.View(contentWidth, contentHeight)
	case PageHelp:
		contentRaw = m.helpModel.View(contentWidth, contentHeight)
	}

	content := styleContentBlock.
		Width(m.width).
		Render(contentRaw)

	return lipgloss.JoinVertical(lipgloss.Left, header, info, content, footer)
}

func (m *Model) contentHeight() int {
	headerLines := lipgloss.Height(m.renderHeader())
	infoLines := lipgloss.Height(m.renderInfoBar())
	footerLines := lipgloss.Height(m.renderFooter())
	available := m.height - headerLines - infoLines - footerLines
	if available < minContentHeight {
		return minContentHeight
	}
	if available > maxContentHeight {
		return maxContentHeight
	}
	return available
}

func (m *Model) renderHeader() string {
	logo := logoWide
	if m.width < 92 {
		logo = logoNarrow
	}

	var b strings.Builder
	b.WriteString(styleAccent.Render(trimToWidth(i18n.Tf("app_header_title", m.config.Version, m.pageTitle()), m.width-2)))
	b.WriteString("\n")
	b.WriteString(sectionDivider(m.width - 2))
	b.WriteString("\n")
	b.WriteString(styleLogo.Render(logo))
	b.WriteString("\n")
	b.WriteString(sectionDivider(m.width - 2))

	return styleHeaderBlock.Width(m.width).Render(b.String())
}

func (m *Model) renderInfoBar() string {
	contextLine := trimToWidth(m.contextLine(), m.width-2)
	if m.showMsg && m.message != "" {
		contextLine = trimToWidth(contextLine+" | "+m.message, m.width-2)
	}

	var warn string
	if !m.cfInstalled {
		warn = styleWarn.Render(i18n.T("cf_not_installed"))
	} else if !m.cfLoggedIn {
		warn = styleWarn.Render(i18n.T("cf_not_logged_in"))
	}

	if warn != "" {
		contextLine = trimToWidth(warn+" | "+contextLine, m.width-2)
	}

	return styleInfoBar.
		Width(m.width).
		Render(contextLine)
}

func (m *Model) renderFooter() string {
	running, stopped, errCount := m.statusSummary()
	statusBar := styleHint.Render(i18n.Tf("footer_status_bar", len(m.tunnels), running, stopped, errCount))
	hint := trimToWidth(m.pageHint(), m.width-2)
	return styleFooterBlock.
		Width(m.width).
		Render(strings.Join([]string{statusBar, sectionDivider(m.width - 2), hint}, "\n"))
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg { return refreshMsg{} }
}

func (m *Model) notify(message string) tea.Cmd {
	return func() tea.Msg { return notifyMsg(message) }
}

func (m *Model) pageTitle() string {
	switch m.page {
	case PageList:
		return i18n.T("page_list")
	case PageCreate:
		return i18n.T("page_create")
	case PageDelete:
		return i18n.T("page_delete")
	case PageLogs:
		return i18n.T("page_logs")
	case PageHelp:
		return i18n.T("page_help")
	default:
		return "TunMan"
	}
}

func (m *Model) statusSummary() (running int, stopped int, errCount int) {
	for _, tunnel := range m.tunnels {
		switch tunnel.Status {
		case model.StatusRunning:
			running++
		case model.StatusError:
			errCount++
		default:
			stopped++
		}
	}
	return running, stopped, errCount
}

func (m *Model) pageHint() string {
	switch m.page {
	case PageList:
		return i18n.T("hint_list")
	case PageCreate:
		return i18n.T("hint_create")
	case PageDelete:
		return i18n.T("hint_delete")
	case PageLogs:
		return i18n.T("hint_logs")
	case PageHelp:
		return i18n.T("hint_help")
	default:
		return i18n.T("hint_default")
	}
}

func (m *Model) contextLine() string {
	switch m.page {
	case PageList:
		return m.listModel.ContextLine(m.tunnels)
	case PageCreate:
		return m.createModel.ContextLine()
	case PageDelete:
		return m.deleteModel.ContextLine(m.tunnels)
	case PageLogs:
		return m.logsModel.ContextLine()
	case PageHelp:
		return i18n.T("ctx_help")
	default:
		return i18n.T("ctx_ready")
	}
}

// 全局快捷键定义
var GlobalKeys = struct {
	Quit    key.Binding
	Back    key.Binding
	Refresh key.Binding
	Help    key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "b"),
		key.WithHelp("esc/b", "back"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

func (m *Model) prepareEditFile(tunnel *model.Tunnel) (string, error) {
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("tunman-edit-%s.json", tunnel.ID))
	data, err := json.MarshalIndent(tunnel, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return "", err
	}
	return tmpPath, nil
}
