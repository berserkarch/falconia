package main

import (
	"falconia/config"
	"falconia/tui"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "don't execute any commands, just simulate")
	flag.Parse()

	// Must run as root for disk operations, unless in dry-run mode
	if os.Getuid() != 0 && !*dryRun {
		fmt.Fprintln(os.Stderr, "falconia must be run as root (sudo or from a live ISO root shell)")
		fmt.Fprintln(os.Stderr, "Use --dry-run for development/testing without root.")
		os.Exit(1)
	}

	app := tui.NewWithConfig(func(cfg *config.InstallConfig) {
		cfg.DryRun = *dryRun
	})

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),       // full-screen TUI
		tea.WithMouseCellMotion(), // optional mouse support
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
