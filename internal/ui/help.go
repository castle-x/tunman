package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/tunman/internal/i18n"
)

// HelpModel 帮助页面
type HelpModel struct{}

func NewHelpModel() HelpModel {
	return HelpModel{}
}

func (m HelpModel) Update(msg tea.Msg, _ int, _ int) (HelpModel, tea.Cmd) {
	return m, nil
}

func (m HelpModel) View(width, height int) string {
	lines := []string{
		styleTitle.Render(i18n.T("help_section_list")),
		i18n.T("help_list_move"),
		i18n.T("help_list_search"),
		i18n.T("help_list_filter"),
		i18n.T("help_list_action"),
		i18n.T("help_list_edit"),
		i18n.T("help_list_add"),
		i18n.T("help_list_delete"),
		i18n.T("help_list_logs"),
		i18n.T("help_list_back"),
		"",
		styleTitle.Render(i18n.T("help_section_create")),
		i18n.T("help_create_move"),
		i18n.T("help_create_edit"),
		i18n.T("help_create_category"),
		i18n.T("help_create_submit"),
		i18n.T("help_create_back"),
		"",
		styleTitle.Render(i18n.T("help_section_delete")),
		i18n.T("help_delete_select"),
		i18n.T("help_delete_mark"),
		i18n.T("help_delete_confirm"),
		i18n.T("help_delete_back"),
		"",
		styleTitle.Render(i18n.T("help_section_logs")),
		i18n.T("help_logs_scroll"),
		i18n.T("help_logs_page"),
		i18n.T("help_logs_jump"),
		i18n.T("help_logs_follow"),
		i18n.T("help_logs_back"),
		"",
		styleTitle.Render(i18n.T("help_section_global")),
		i18n.T("help_global_quit"),
		i18n.T("help_global_refresh"),
	}

	return strings.Join(fitLines(lines, height, width), "\n")
}
