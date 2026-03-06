package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/tunman/internal/core"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/model"
)

const createLabelWidth = 14

type CreateModel struct {
	cursor  int
	editing bool
	exit    bool
	busy    bool

	category model.Category

	baseDomainInput textinput.Model
	prefixInput     textinput.Model
	portInput       textinput.Model
	descInput       textinput.Model

	message string
}

func NewCreateModel() CreateModel {
	mkInput := func(placeholder string, charLimit int, width int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.Prompt = ""
		ti.CharLimit = charLimit
		ti.Width = width
		return ti
	}

	port := mkInput(i18n.T("create_placeholder_port"), 5, 10)
	port.SetValue("3000")

	return CreateModel{
		category:        model.CategoryCustom,
		baseDomainInput: mkInput(i18n.T("create_placeholder_base_domain"), 128, 48),
		prefixInput:     mkInput(i18n.T("create_placeholder_prefix"), 64, 40),
		portInput:       port,
		descInput:       mkInput(i18n.T("create_placeholder_desc"), 120, 48),
	}
}

func (m CreateModel) fields() []createField {
	fs := []createField{
		{kind: fieldCategory, id: fidCategory, label: i18n.T("create_field_category")},
	}

	switch m.category {
	case model.CategoryCustom:
		fs = append(fs,
			createField{kind: fieldInput, id: fidBaseDomain, label: i18n.T("create_field_base_domain"), input: &m.baseDomainInput},
			createField{kind: fieldInput, id: fidPrefix, label: i18n.T("create_field_prefix"), input: &m.prefixInput},
			createField{kind: fieldInput, id: fidPort, label: i18n.T("create_field_port"), input: &m.portInput},
			createField{kind: fieldInput, id: fidDesc, label: i18n.T("create_field_desc"), input: &m.descInput},
		)
	case model.CategoryTesting:
		fs = append(fs,
			createField{kind: fieldInput, id: fidBaseDomain, label: i18n.T("create_field_base_domain"), input: &m.baseDomainInput},
			createField{kind: fieldInput, id: fidPort, label: i18n.T("create_field_port"), input: &m.portInput},
			createField{kind: fieldInput, id: fidDesc, label: i18n.T("create_field_desc"), input: &m.descInput},
		)
	case model.CategoryEphemeral:
		fs = append(fs,
			createField{kind: fieldInput, id: fidPort, label: i18n.T("create_field_port"), input: &m.portInput},
			createField{kind: fieldInput, id: fidDesc, label: i18n.T("create_field_desc"), input: &m.descInput},
		)
	}

	fs = append(fs, createField{kind: fieldSubmit, id: fidSubmit, label: i18n.T("create_action_submit")})
	return fs
}

type fieldKind int

const (
	fieldCategory fieldKind = iota
	fieldInput
	fieldSubmit
)

type fieldID int

const (
	fidCategory fieldID = iota
	fidBaseDomain
	fidPrefix
	fidPort
	fidDesc
	fidSubmit
)

type createField struct {
	kind  fieldKind
	id    fieldID
	label string
	input *textinput.Model
}

// tunnelSetupMsg is sent back after async SetupTunnel completes.
type tunnelSetupMsg struct {
	tunnel model.Tunnel
	err    error
}

func (m CreateModel) Update(msg tea.Msg, storage *core.Storage, controller *core.Controller, _ int, _ int) (CreateModel, tea.Cmd) {
	if setupMsg, ok := msg.(tunnelSetupMsg); ok {
		m.busy = false
		if setupMsg.err != nil {
			m.message = i18n.Tf("create_failed", setupMsg.err.Error())
			return m, notifyCmd(m.message)
		}
		if err := storage.AddTunnel(setupMsg.tunnel); err != nil {
			m.message = i18n.Tf("create_failed", err.Error())
			return m, notifyCmd(m.message)
		}
		t := setupMsg.tunnel
		if err := controller.Start(&t); err == nil {
			_ = storage.UpdateTunnel(t)
		}
		name := t.Name
		m = NewCreateModel()
		return m, tunnelCreatedCmd(name)
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.busy {
		return m, nil
	}

	fs := m.fields()
	maxIdx := len(fs) - 1

	if m.editing {
		switch keyMsg.String() {
		case "enter", "esc":
			m.editing = false
			m.blurInputs()
			return m, nil
		default:
			return m.updateFocusedInput(keyMsg)
		}
	}

	switch keyMsg.String() {
	case "up", "k":
		m.cursor--
		if m.cursor < 0 {
			m.cursor = maxIdx
		}
		m.blurInputs()
		return m, nil

	case "down", "j", "tab":
		m.cursor++
		if m.cursor > maxIdx {
			m.cursor = 0
		}
		m.blurInputs()
		return m, nil

	case "left", "h":
		if m.cursor < len(fs) && fs[m.cursor].kind == fieldCategory {
			m.category = prevCategory(m.category)
			m.clampCursor(fs)
		}
		return m, nil

	case "right", "l":
		if m.cursor < len(fs) && fs[m.cursor].kind == fieldCategory {
			m.category = nextCategory(m.category)
			m.clampCursor(fs)
		}
		return m, nil

	case "esc", "b":
		m.exit = true
		m.message = ""
		m.blurInputs()
		return m, nil

	case "ctrl+s":
		return m.submit(storage, controller)

	case "enter", " ":
		if m.cursor >= len(fs) {
			return m, nil
		}
		f := fs[m.cursor]
		switch f.kind {
		case fieldCategory:
			m.category = nextCategory(m.category)
			m.clampCursor(fs)
		case fieldInput:
			m.editing = true
			m.focusByIndex(m.cursor, fs)
		case fieldSubmit:
			return m.submit(storage, controller)
		}
		return m, nil
	}

	return m, nil
}

func (m CreateModel) View(width, height int) string {
	fs := m.fields()
	lines := make([]string, 0, height)

	for i, f := range fs {
		switch f.kind {
		case fieldCategory:
			val := fmt.Sprintf("%s  %s", m.category.DisplayName(), m.category.Icon())
			if i == m.cursor {
				val += styleHint.Render("  ←/→")
			}
			lines = append(lines, m.fieldRow(i, f.label, val, width))

		case fieldInput:
			val := m.inputView(f.id)
			if i == m.cursor && m.editing {
				val += styleHint.Render("  " + i18n.T("create_editing"))
			}
			lines = append(lines, m.fieldRow(i, f.label, val, width))

		case fieldSubmit:
			lines = append(lines, "")
			prefix := "  "
			if m.busy {
				lines = append(lines, styleHint.Render(trimToWidth("  "+i18n.T("create_setting_up"), width)))
			} else if i == m.cursor {
				prefix = styleCursor.Render("❯") + " "
				lines = append(lines, styleSelectedRow.Render(trimToWidth(prefix+"["+f.label+"]", width)))
			} else {
				lines = append(lines, styleSelectable.Render(trimToWidth(prefix+"["+f.label+"]", width)))
			}
		}
	}

	if m.message != "" {
		lines = append(lines, styleWarn.Render(trimToWidth(m.message, width)))
	}

	return strings.Join(fitLines(lines, height, width), "\n")
}

func (m CreateModel) inputView(id fieldID) string {
	switch id {
	case fidBaseDomain:
		return m.baseDomainInput.View()
	case fidPrefix:
		return m.prefixInput.View()
	case fidPort:
		return m.portInput.View()
	case fidDesc:
		return m.descInput.View()
	default:
		return ""
	}
}

func (m CreateModel) fieldRow(index int, label, value string, width int) string {
	prefix := "  "
	if index == m.cursor {
		prefix = styleCursor.Render("❯") + " "
	}
	paddedLabel := padRight(label+":", createLabelWidth)
	row := prefix + paddedLabel + " " + value
	if index == m.cursor {
		return styleSelectedRow.Render(trimToWidth(row, width))
	}
	return styleNormalRow.Render(trimToWidth(row, width))
}

func (m CreateModel) ContextLine() string {
	switch m.category {
	case model.CategoryCustom:
		prefix := strings.TrimSpace(m.prefixInput.Value())
		domain := strings.TrimSpace(m.baseDomainInput.Value())
		if prefix != "" && domain != "" {
			return fmt.Sprintf("%s — %s.%s", m.category.DisplayName(), prefix, domain)
		}
		return m.category.DisplayName()
	case model.CategoryTesting:
		domain := strings.TrimSpace(m.baseDomainInput.Value())
		if domain != "" {
			return fmt.Sprintf("%s — *.%s", m.category.DisplayName(), domain)
		}
		return m.category.DisplayName()
	default:
		port := strings.TrimSpace(m.portInput.Value())
		if port != "" {
			return fmt.Sprintf("%s — :%s", m.category.DisplayName(), port)
		}
		return m.category.DisplayName()
	}
}

func (m *CreateModel) ConsumeExit() bool {
	if !m.exit {
		return false
	}
	m.exit = false
	return true
}

func (m CreateModel) submit(storage *core.Storage, controller *core.Controller) (CreateModel, tea.Cmd) {
	if err := m.validate(); err != nil {
		m.message = err.Error()
		return m, notifyCmd(m.message)
	}

	if m.category != model.CategoryEphemeral {
		if err := controller.CheckCloudflared(); err != nil {
			m.message = i18n.T("create_err_cf_not_installed")
			return m, notifyCmd(m.message)
		}
		if err := controller.CheckAuth(); err != nil {
			m.message = i18n.T("create_err_cf_not_logged_in")
			return m, notifyCmd(m.message)
		}
	}

	port, _ := strconv.Atoi(strings.TrimSpace(m.portInput.Value()))

	tunnel := model.Tunnel{
		Category:    m.category,
		Description: strings.TrimSpace(m.descInput.Value()),
		Port:        port,
		Status:      model.StatusStopped,
	}

	switch m.category {
	case model.CategoryCustom:
		tunnel.BaseDomain = strings.TrimSpace(m.baseDomainInput.Value())
		tunnel.Prefix = strings.TrimSpace(m.prefixInput.Value())
		tunnel.Name = tunnel.Prefix
		tunnel.ID = buildTunnelRecordID(tunnel.Name)

	case model.CategoryTesting:
		tunnel.BaseDomain = strings.TrimSpace(m.baseDomainInput.Value())
		tunnel.Prefix = model.GenerateTestingPrefix()
		tunnel.Name = fmt.Sprintf("test-%s", tunnel.Prefix)
		tunnel.ID = buildTunnelRecordID(tunnel.Name)

	case model.CategoryEphemeral:
		tunnel.Name = fmt.Sprintf("quick-%d", port)
		tunnel.ID = buildTunnelRecordID(tunnel.Name)
	}

	if tunnel.Category == model.CategoryEphemeral {
		if err := storage.AddTunnel(tunnel); err != nil {
			m.message = i18n.Tf("create_failed", err.Error())
			return m, notifyCmd(m.message)
		}
		name := tunnel.Name
		m = NewCreateModel()
		return m, tunnelCreatedCmd(name)
	}

	m.busy = true
	m.message = i18n.T("create_setting_up")
	return m, setupTunnelCmd(tunnel, controller)
}

func setupTunnelCmd(tunnel model.Tunnel, controller *core.Controller) tea.Cmd {
	return func() tea.Msg {
		err := controller.SetupTunnel(&tunnel)
		return tunnelSetupMsg{tunnel: tunnel, err: err}
	}
}

func (m CreateModel) validate() error {
	port, err := strconv.Atoi(strings.TrimSpace(m.portInput.Value()))
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf(i18n.T("create_err_port_range"))
	}

	switch m.category {
	case model.CategoryCustom:
		if strings.TrimSpace(m.baseDomainInput.Value()) == "" {
			return fmt.Errorf(i18n.T("create_err_base_domain_required"))
		}
		if strings.TrimSpace(m.prefixInput.Value()) == "" {
			return fmt.Errorf(i18n.T("create_err_prefix_required"))
		}
	case model.CategoryTesting:
		if strings.TrimSpace(m.baseDomainInput.Value()) == "" {
			return fmt.Errorf(i18n.T("create_err_base_domain_required"))
		}
	}
	return nil
}

func (m CreateModel) updateFocusedInput(keyMsg tea.KeyMsg) (CreateModel, tea.Cmd) {
	fs := m.fields()
	if m.cursor >= len(fs) {
		return m, nil
	}
	f := fs[m.cursor]
	if f.kind != fieldInput {
		return m, nil
	}

	var cmd tea.Cmd
	switch f.id {
	case fidBaseDomain:
		m.baseDomainInput, cmd = m.baseDomainInput.Update(keyMsg)
	case fidPrefix:
		m.prefixInput, cmd = m.prefixInput.Update(keyMsg)
	case fidPort:
		m.portInput, cmd = m.portInput.Update(keyMsg)
	case fidDesc:
		m.descInput, cmd = m.descInput.Update(keyMsg)
	}
	return m, cmd
}

func (m *CreateModel) blurInputs() {
	m.baseDomainInput.Blur()
	m.prefixInput.Blur()
	m.portInput.Blur()
	m.descInput.Blur()
}

func (m *CreateModel) focusByIndex(idx int, fs []createField) {
	m.blurInputs()
	if idx >= len(fs) {
		return
	}
	f := fs[idx]
	if f.kind != fieldInput {
		return
	}
	switch f.id {
	case fidBaseDomain:
		m.baseDomainInput.Focus()
	case fidPrefix:
		m.prefixInput.Focus()
	case fidPort:
		m.portInput.Focus()
	case fidDesc:
		m.descInput.Focus()
	}
}

func (m *CreateModel) clampCursor(fs []createField) {
	newFS := m.fields()
	if m.cursor >= len(newFS) {
		m.cursor = len(newFS) - 1
	}
	_ = fs
}

func buildTunnelRecordID(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	base = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "tunnel"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}

func nextCategory(current model.Category) model.Category {
	switch current {
	case model.CategoryCustom:
		return model.CategoryTesting
	case model.CategoryTesting:
		return model.CategoryEphemeral
	default:
		return model.CategoryCustom
	}
}

func prevCategory(current model.Category) model.Category {
	switch current {
	case model.CategoryCustom:
		return model.CategoryEphemeral
	case model.CategoryTesting:
		return model.CategoryCustom
	default:
		return model.CategoryTesting
	}
}

func tunnelCreatedCmd(name string) tea.Cmd {
	return tea.Batch(
		refreshCmd(),
		notifyCmd(i18n.Tf("create_success", name)),
		func() tea.Msg { return tunnelCreatedMsg{Name: name} },
	)
}
