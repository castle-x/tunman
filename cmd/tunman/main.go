package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/ui"
)

var (
	Version   = "dev"
	BuildTime = ""
)

func main() {
	i18n.MustInit()

	if len(os.Args) > 1 {
		// CLI 模式
		runCLI(os.Args[1:])
		return
	}

	// TUI 模式
	model, err := ui.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.Tf("err_app_runtime", err)+"\n")
		os.Exit(1)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, i18n.Tf("err_app_runtime", err)+"\n")
		os.Exit(1)
	}
}

func runCLI(args []string) {
	switch args[0] {
	case "list", "ls":
		fmt.Println(i18n.T("cmd_list_wip"))
	case "start":
		if len(args) < 2 {
			fmt.Println(i18n.T("cmd_start_usage"))
			return
		}
		fmt.Printf(i18n.Tf("cmd_starting", args[1]) + "\n")
	case "stop":
		if len(args) < 2 {
			fmt.Println(i18n.T("cmd_stop_usage"))
			return
		}
		fmt.Printf(i18n.Tf("cmd_stopping", args[1]) + "\n")
	case "status":
		fmt.Println(i18n.T("cmd_status_wip"))
	case "version", "v":
		fmt.Printf("TunMan %s", Version)
		if BuildTime != "" {
			fmt.Printf(" (%s)", BuildTime)
		}
		fmt.Println()
	default:
		fmt.Printf(i18n.Tf("err_cmd_unknown", args[0]) + "\n")
		fmt.Println(i18n.T("cmd_usage"))
	}
}
