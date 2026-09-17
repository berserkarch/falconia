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
	var exclude packageList
	flag.Var(&exclude, "exclude", "comma-separated packages to leave out of every install step (repeatable)")
	flag.Parse()

	// Must run as root for disk operations, unless in dry-run mode
	if os.Getuid() != 0 && !*dryRun {
		fmt.Fprintln(os.Stderr, "falconia must be run as root (sudo or from a live ISO root shell)")
		fmt.Fprintln(os.Stderr, "Use --dry-run for development/testing without root.")
		os.Exit(1)
	}

	var app tui.App
	if *resume {
		app = resumeApp(*dryRun, exclude)
	} else {
		hw := installer.DetectHardware()
		app = tui.NewWithConfig(func(cfg *config.InstallConfig) {
			cfg.DryRun = *dryRun
			cfg.Hardware = hw
			installer.MergeExcludes(cfg, exclude)
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
func resumeApp(dryRun bool, exclude []string) tui.App {
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
	installer.MergeExcludes(cfg, exclude)
	if len(cfg.ExcludePackages) > 0 {
		fmt.Println("\nExcluded packages: " + strings.Join(cfg.ExcludePackages, " "))
	}
	fmt.Print("\nContinue? [y/N] ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		os.Exit(0)
	}
	// Persist new exclusions right away so they survive another interruption.
	if err := installer.SaveState(state); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save resume state: %v\n", err)
	}
	return tui.NewResume(state)
}

// packageList collects --exclude values; each may be comma-separated.
type packageList []string

func (l *packageList) String() string { return strings.Join(*l, ",") }

func (l *packageList) Set(v string) error {
	for _, pkg := range strings.Split(v, ",") {
		if pkg = strings.TrimSpace(pkg); pkg != "" {
			*l = append(*l, pkg)
		}
	}
	return nil
}

func promptPassphrase(dev string) (string, error) {
	fmt.Printf("LUKS passphrase for %s (empty to skip): ", dev)
	pass, err := term.ReadPassword(os.Stdin.Fd())
	fmt.Println()
	return string(pass), err
}
