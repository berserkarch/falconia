package steps

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"falconia/config"
	"falconia/data"
	"falconia/installer"
	"falconia/style"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Install step definitions ─────────────────────────────────────────────────

type installStep struct {
	label string
	soft  bool // if true, failure logs a warning and continues rather than aborting
	run   func(*config.InstallConfig, installer.LineHandler) error
}

// stepRegistry maps each StepKey from data.Pipeline to its implementation.
// To add a new step: declare the key in data/flow.go, add it to data.Pipeline,
// then register the implementation here.
var stepRegistry = map[data.StepKey]func(*config.InstallConfig, installer.LineHandler) error{
	data.StepVerifyInternet: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.CheckInternet(log)
	},
	data.StepSyncClock: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.SyncClock(log)
	},
	data.StepRankMirrors: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.RankMirrors(c, log)
	},
	data.StepPartitionDisk: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.PartitionDisk(c, log)
	},
	data.StepFormatDisks: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.FormatDisks(c, log)
	},
	data.StepMountDisks: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.MountDisks(c, log)
	},
	data.StepCopyLiveFiles: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.CopyLiveFiles(c, log)
	},
	data.StepPacstrap: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.Pacstrap(c, log)
	},
	data.StepGenFstab: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.GenFstab(c, log)
	},
	data.StepSetTimezone: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.SetTimezone(c, log)
	},
	data.StepWriteCrypttab: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.GenCrypttab(c, log)
	},
	data.StepSetLocale: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.SetLocale(c, log)
	},
	data.StepConfigNetwork: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.ConfigureNetwork(c, log)
	},
	data.StepSetHostname: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.SetHostname(c, log)
	},
	data.StepRootPassword: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.SetRootPassword(c, log)
	},
	data.StepCreateUsers: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.CreateUsers(c, log)
	},
	data.StepBootloader: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.InstallBootloader(c, log)
	},
	data.StepInstallDesktop: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.InstallDesktop(c, log)
	},
	data.StepInstallPkgs: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.InstallPackages(c, log)
	},
	data.StepInstallDrivers: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.InstallDrivers(c, log)
	},
	data.StepEnableServices: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.EnableServices(c, log)
	},
	data.StepCleanup: func(c *config.InstallConfig, log installer.LineHandler) error {
		return installer.Cleanup(c, log)
	},
}

func buildSteps(cfg *config.InstallConfig) []installStep {
	var steps []installStep
	for _, def := range data.Pipeline {
		if def.When != nil && !def.When(cfg) {
			continue
		}
		fn, ok := stepRegistry[def.Key]
		if !ok {
			panic("no implementation registered for step: " + string(def.Key))
		}
		steps = append(steps, installStep{label: def.Label, soft: def.Soft, run: fn})
	}
	return steps
}

// ── Messages ──────────────────────────────────────────────────────────────────

// LogLineMsg carries a single line of installer output.
type LogLineMsg string

// StepCompleteMsg signals install step N finished successfully.
type StepCompleteMsg int

// InstallErrorMsg signals a fatal install error.
type InstallErrorMsg struct {
	Step string
	Err  error
}

// InstallDoneMsg signals all steps completed.
type InstallDoneMsg struct{}

// ── Model ─────────────────────────────────────────────────────────────────────

type progressState int

const (
	stateRunning progressState = iota
	stateDone
	stateError
)

// ProgressModel is the Phase 2 model.
// A goroutine runs all installer steps sequentially, piping output and step
// events through a buffered channel. waitForMsg drains it one message at a
// time back into the bubbletea update loop for live streaming.
type ProgressModel struct {
	cfg     *config.InstallConfig
	steps   []installStep
	current int
	state   progressState

	ch       chan tea.Msg
	logLines []string
	viewport viewport.Model

	errStep string
	errMsg  string
	logFile *os.File
}

func NewProgress(cfg *config.InstallConfig) ProgressModel {
	vp := viewport.New(80, 12)
	// Initialize channel here so it's shared across all value copies of this struct.
	// Channels are reference types, so copies of ProgressModel all share the same channel.
	ch := make(chan tea.Msg, 256)
	return ProgressModel{
		cfg:      cfg,
		steps:    buildSteps(cfg),
		viewport: vp,
		ch:       ch,
	}
}

// Init launches the installer goroutine and begins draining the channel.
func (m ProgressModel) Init() tea.Cmd {
	// Open log file for Phase 2.
	f, err := os.OpenFile("falconia-install.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		m.logFile = f
		f.WriteString(fmt.Sprintf("\n--- Installation started at %s ---\n", time.Now().Format(time.RFC3339)))
		if m.cfg.DryRun {
			f.WriteString("--- DRY RUN MODE ENABLED ---\n")
		}
	}

	go func() {
		defer func() {
			if m.logFile != nil {
				m.logFile.Close()
			}
		}()

		for i, step := range m.steps {
			logHandler := func(line string) {
				m.ch <- LogLineMsg(line)
				if m.logFile != nil {
					// Strip ANSI escape codes before writing to log file
					cleanLine := stripANSI(line)
					m.logFile.WriteString(cleanLine + "\n")
				}
			}
			if err := step.run(m.cfg, logHandler); err != nil {
				if step.soft {
					logHandler("\033[33m⚠  " + step.label + " failed (non-fatal): " + err.Error() + " — continuing\033[0m")
				} else {
					m.ch <- InstallErrorMsg{Step: step.label, Err: err}
					return
				}
			}
			m.ch <- StepCompleteMsg(i)
		}
		m.ch <- InstallDoneMsg{}
	}()

	return waitForMsg(m.ch)
}

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[-a-zA-Z\\d\\/#&.:=?%@~]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-ntqry=><~]))")

func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// waitForMsg returns a Cmd that blocks until the next message arrives on ch.
func waitForMsg(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 26 // 20 padding + small gap
		m.viewport.Height = 12

	case LogLineMsg:
		m.logLines = append(m.logLines, string(msg))
		m.refreshViewport()
		return m, waitForMsg(m.ch)

	case StepCompleteMsg:
		m.current = int(msg) + 1
		return m, waitForMsg(m.ch)

	case InstallErrorMsg:
		m.state = stateError
		m.errStep = msg.Step
		m.errMsg = msg.Err.Error()
		return m, nil

	case InstallDoneMsg:
		m.state = stateDone
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.state != stateRunning {
				return m, tea.Quit
			}
		case "r":
			if m.state == stateDone {
				return m, installer.RebootCmd(m.cfg.DryRun)
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *ProgressModel) refreshViewport() {
	var sb strings.Builder
	for _, line := range m.logLines {
		sb.WriteString(style.StyleLogLine.Render(line) + "\n")
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func (m ProgressModel) View() string {
	switch m.state {
	case stateDone:
		return m.viewDone()
	case stateError:
		return m.viewError()
	default:
		return m.viewRunning()
	}
}

func (m ProgressModel) viewRunning() string {
	var b strings.Builder
	b.WriteString(style.StyleTitle.Render("INSTALLING BERSERKARCH") + "\n\n")

	for i, s := range m.steps {
		var icon string
		switch {
		case i < m.current:
			icon = style.StyleGood.Render("[✓]")
		case i == m.current:
			icon = style.StyleBlue().Render("[▶]")
		default:
			icon = style.StyleMuted.Render("[ ]")
		}

		label := s.label
		if i == m.current {
			label = style.StyleSelected.Render(label)
		} else if i < m.current {
			label = style.StyleMuted.Render(label)
		}
		b.WriteString(fmt.Sprintf("%s  %s\n", icon, label))
	}

	b.WriteString("\n")

	pct := 0.0
	if len(m.steps) > 0 {
		pct = float64(m.current) / float64(len(m.steps))
	}
	b.WriteString(style.ProgressBar(60, pct) +
		style.StyleMuted.Render(fmt.Sprintf("  %d%%\n\n", int(pct*100))))

	if len(m.logLines) > 0 {
		b.WriteString(m.viewport.View() + "\n")
	}

	return b.String()
}

func (m ProgressModel) viewDone() string {
	var b strings.Builder
	b.WriteString(style.StyleGood.Render("✓  Installation complete!") + "\n\n")
	b.WriteString(style.StyleValue.Render("BerserkArch has been installed successfully.") + "\n")

	if m.cfg.DryRun {
		b.WriteString(style.StyleMuted.Render("Dry-run mode: no changes were actually made to your disk.") + "\n\n")
		b.WriteString(style.StyleButtonActive.Render("EXIT") + "\n\n")
		b.WriteString(style.HelpRow("r/q", "exit to shell"))
	} else {
		b.WriteString(style.StyleMuted.Render("Remove the installation media and reboot.") + "\n\n")
		b.WriteString(style.StyleButtonActive.Render("REBOOT") + "\n\n")
		b.WriteString(style.HelpRow("r", "reboot now", "q", "quit to shell"))
	}
	return b.String()
}

func (m ProgressModel) viewError() string {
	var b strings.Builder
	b.WriteString(style.StyleError.Render("✗  Installation failed") + "\n\n")
	b.WriteString(style.StyleKey.Render("Failed step:  ") + style.StyleValue.Render(m.errStep) + "\n")
	b.WriteString(style.StyleKey.Render("Error:        ") + style.StyleError.Render(m.errMsg) + "\n\n")

	if len(m.logLines) > 0 {
		b.WriteString(style.StyleSubtitle.Render("Full log:") + "\n")
		b.WriteString(m.viewport.View() + "\n")
	}

	b.WriteString(style.HelpRow("↑↓", "scroll log", "q", "exit"))
	return b.String()
}
