package tui

import (
	"falconia/config"
	"falconia/installer"
	"falconia/style"
	"falconia/tui/steps"
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// phase tracks which phase the installer is in.
type phase int

const (
	phase1 phase = iota // configuration
	phase2              // installation
)

// stepID names each Phase 1 step in order.
type stepID int

const (
	stepWelcome stepID = iota
	stepDisk
	stepPartitions
	stepNetwork
	stepLocale
	stepHostname
	stepUsers
	stepKernel
	stepDesktop
	stepPackages
	stepBootloader
	stepConfirm
	stepCount
)

// App is the root bubbletea model.
type App struct {
	phase    phase
	cfg      *config.InstallConfig
	advanced bool
	quitting bool

	// Phase 1
	step    stepID
	current tea.Model

	// Phase 2
	progress steps.ProgressModel

	width  int
	height int
}

// New creates the root App model, auto-detecting firmware.
func New() App {
	return NewWithConfig(nil)
}

// NewWithConfig creates the root App model and applies a callback to the config.
func NewWithConfig(fn func(*config.InstallConfig)) App {
	cfg := config.Defaults()
	cfg.Firmware = installer.DetectFirmware()
	if fn != nil {
		fn(cfg)
	}

	app := App{cfg: cfg}
	app.loadStep(stepWelcome)
	return app
}

func (a *App) loadStep(id stepID) {
	a.step = id
	switch id {
	case stepWelcome:
		a.current = steps.NewWelcome(a.cfg)
	case stepDisk:
		a.current = steps.NewDisk(a.cfg, a.advanced)
	case stepPartitions:
		a.current = steps.NewPartitionMapping(a.cfg)
	case stepNetwork:
		a.current = steps.NewNetwork(a.cfg)
	case stepLocale:
		a.current = steps.NewLocale(a.cfg)
	case stepHostname:
		a.current = steps.NewHostname(a.cfg)
	case stepUsers:
		a.current = steps.NewUsers(a.cfg)
	case stepKernel:
		a.current = steps.NewKernel(a.cfg)
	case stepDesktop:
		a.current = steps.NewDesktop(a.cfg)
	case stepPackages:
		a.current = steps.NewPackages(a.cfg, a.advanced)
	case stepBootloader:
		a.current = steps.NewBootloader(a.cfg)
	case stepConfirm:
		a.current = steps.NewConfirm(a.cfg)
	}
}

func (a App) Init() tea.Cmd {
	if a.current != nil {
		return a.current.Init()
	}
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		if a.phase == phase2 {
			updated, cmd := a.progress.Update(msg)
			a.progress = updated.(steps.ProgressModel)
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		if a.quitting {
			switch msg.String() {
			case "y", "enter":
				return a, tea.Quit
			case "n", "esc", "q":
				a.quitting = false
				return a, nil
			}
			return a, nil
		}

		// Global quit
		if msg.String() == "q" {
			if a.phase == phase1 {
				a.quitting = true
				return a, nil
			}
			// In phase 2, let the progress model handle 'q' (which typically quits only when not running)
		}

		if msg.String() == "ctrl+c" {
			a.quitting = true
			return a, nil
		}

		// Global advanced mode toggle
		if a.phase == phase1 && msg.String() == "ctrl+a" {
			a.advanced = !a.advanced
			if s, ok := a.current.(steps.Saveable); ok {
				s.Save()
			}
			a.loadStep(a.step)
			return a, a.current.Init()
		}

	// ── Phase 1 navigation ──────────────────────────────────────────────────

	case steps.DoneMsg:
		next := a.step + 1
		if next == stepPartitions && a.cfg.PartitionScheme == "guided" {
			next++
		}
		if next < stepCount {
			a.loadStep(next)
			return a, a.current.Init()
		}
		return a, nil

	case steps.BackMsg:
		prev := a.step - 1
		if prev == stepPartitions && a.cfg.PartitionScheme == "guided" {
			prev--
		}
		if prev >= stepWelcome {
			a.loadStep(prev)
			return a, a.current.Init()
		}
		return a, nil

	case steps.LaunchCfdiskMsg:
		c := exec.Command("cfdisk", msg.Disk)
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				// Handle error? For now just proceed as if done.
				return steps.DoneMsg{}
			}
			return steps.DoneMsg{}
		})

	case steps.StartInstallMsg:
		// Switch to Phase 2
		a.phase = phase2
		a.progress = steps.NewProgress(a.cfg)
		return a, a.progress.Init()
	}

	// Block input if quitting
	if a.quitting {
		return a, nil
	}

	// Delegate to active child model
	if a.phase == phase2 {
		updated, cmd := a.progress.Update(msg)
		a.progress = updated.(steps.ProgressModel)
		return a, cmd
	}

	updated, cmd := a.current.Update(msg)
	a.current = updated
	return a, cmd
}

const (
	minWidth  = 80
	minHeight = 24
)

func (a App) View() string {
	if a.width < minWidth || a.height < minHeight {
		return a.renderSizeWarning()
	}

	var view string
	if a.phase == phase2 {
		view = style.StyleMain.Render(a.progress.View())
	} else {
		view = a.viewPhase1()
	}

	if a.quitting {
		return a.renderQuitConfirmation(view)
	}

	return view
}

func (a App) renderSizeWarning() string {
	msg := fmt.Sprintf(
		"%s\n\nYour terminal is too small (%dx%d).\nPlease resize or maximize to at least %dx%d.",
		style.StyleError.Render("TERMINAL TOO SMALL"),
		a.width, a.height,
		minWidth, minHeight,
	)

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(1, 4).
			Align(lipgloss.Center).
			Render(msg),
	)
}

func (a App) renderQuitConfirmation(baseView string) string {
	dialog := lipgloss.NewStyle().
		Width(50).
		Height(8).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")).
		Padding(1, 2).
		Align(lipgloss.Center, lipgloss.Center).
		Render(
			lipgloss.JoinVertical(lipgloss.Center,
				style.StyleError.Render("QUIT INSTALLATION?"),
				"\nAre you sure you want to exit?\nAll progress will be lost.\n",
				style.HelpRow("y", "yes, quit", "n", "no, stay"),
			),
		)

	// Center the dialog over the base view
	return style.PlaceOverlay(a.width, a.height, dialog, baseView)
}

func (a App) viewPhase1() string {
	mainContent := a.current.View()
	mainPane := style.StyleMain.Render(mainContent)

	// If terminal is too narrow, hide the sidebar
	if a.width < 150 {
		return mainPane
	}

	sidebar := SidebarView(a.cfg, int(a.step)+1, int(stepCount), a.advanced)

	mw := lipgloss.Width(mainPane)
	sw := lipgloss.Width(sidebar)

	// If they don't fit side-by-side, hide the sidebar
	if a.width < mw+sw+4 {
		return mainPane
	}

	// Ensure the main pane is a perfect rectangle before joining.
	// We use lipgloss.NewStyle().Width() to pad shorter lines with spaces.
	mainPaneRect := lipgloss.NewStyle().Width(mw).Render(mainPane)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		mainPaneRect,
		lipgloss.PlaceHorizontal(a.width-mw, lipgloss.Right, sidebar),
	)
}
