package main

import (
	"bufio"
	"falconia/config"
	"falconia/installer"
	"falconia/tui"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "don't execute any commands, just simulate")
	resume := flag.Bool("resume", false, "continue an interrupted installation")
	flag.Parse()

	// Must run as root for disk operations, unless in dry-run mode
	if os.Getuid() != 0 && !*dryRun {
		fmt.Fprintln(os.Stderr, "falconia must be run as root (sudo or from a live ISO root shell)")
		fmt.Fprintln(os.Stderr, "Use --dry-run for development/testing without root.")
		os.Exit(1)
	}

	var app tui.App
	if *resume {
		app = resumeApp(*dryRun)
	} else {
		hw := installer.DetectHardware()
		app = tui.NewWithConfig(func(cfg *config.InstallConfig) {
			cfg.DryRun = *dryRun
			cfg.Hardware = hw
		})
	}

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(), // full-screen TUI
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// resumeApp loads the interrupted install, shows what is left and asks for
// confirmation before handing over to the TUI.
func resumeApp(dryRun bool) tui.App {
	logLine := func(line string) { fmt.Println(line) }
	state, err := installer.FindResumeState(dryRun, promptPassphrase, logLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot resume: %v\n", err)
		os.Exit(1)
	}

	cfg := state.Config
	pending := state.PendingSteps()
	fmt.Printf("\nInterrupted install on %s (saved %s)\n", cfg.Disk, state.UpdatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("%d step(s) done, %d remaining:\n", len(state.Completed), len(pending))
	for _, label := range pending {
		fmt.Println("  - " + label)
	}
	fmt.Print("\nContinue? [y/N] ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		os.Exit(0)
	}
	return tui.NewResume(state)
}

func promptPassphrase(dev string) (string, error) {
	fmt.Printf("LUKS passphrase for %s (empty to skip): ", dev)
	pass, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Println()
	return string(pass), err
}
