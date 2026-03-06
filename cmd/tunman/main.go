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
		os.Exit(runCLI(os.Args[1:]))
	}

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
