package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"laxy-services/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println("laxy-services", version)
		return
	}
	p := tea.NewProgram(tui.New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
