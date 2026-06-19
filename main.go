package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/umit144/ssh-matrix/internal/ssh"
	"github.com/umit144/ssh-matrix/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("ssh-matrix " + version)
		return
	}

	hosts, err := ssh.ParseConfig()
	if err != nil {
		hosts = nil
	}

	p := tea.NewProgram(tui.New(hosts),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
