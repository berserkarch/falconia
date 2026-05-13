package steps

import (
	"strings"

	"falconia/config"
	"falconia/style"

	tea "github.com/charmbracelet/bubbletea"
)

type postOption struct {
	label string
	desc  string
	get   func() bool
	set   func(bool)
}

// PostInstallModel handles post-install service/feature toggles.
type PostInstallModel struct {
	cfg     *config.InstallConfig
	options []postOption
	cursor  int
}

func NewPostInstall(cfg *config.InstallConfig) PostInstallModel {
	opts := []postOption{
		{
			"Rank mirrors (reflector)",
			"Run reflector to sort mirrors by speed before pacstrap",
			func() bool { return cfg.RankMirrors },
			func(v bool) { cfg.RankMirrors = v },
		},
		{
			"Enable NetworkManager",
			"Always enabled — required for networking after reboot",
			func() bool { return true },
			func(bool) {}, // always on, not togglable
		},
		{
			"Enable SSH (sshd)",
			"Start OpenSSH server on boot",
			func() bool { return cfg.EnableSSH },
			func(v bool) { cfg.EnableSSH = v },
		},
		{
			"Enable Bluetooth",
			"Start bluetooth.service on boot",
			func() bool { return cfg.EnableBluetooth },
			func(v bool) { cfg.EnableBluetooth = v },
		},
		{
			"Enable CUPS (printing)",
			"Start CUPS printing system on boot",
			func() bool { return cfg.EnableCups },
			func(v bool) { cfg.EnableCups = v },
		},
	}
	return PostInstallModel{cfg: cfg, options: opts}
}

func (m PostInstallModel) Init() tea.Cmd { return nil }

func (m PostInstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "up", "k", "shift+tab":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}
		case "down", "j", "tab":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case " ":
			opt := m.options[m.cursor]
			// skip the always-on NetworkManager row
			if opt.label != "Enable NetworkManager" {
				opt.set(!opt.get())
			}
			return m, nil
		case "enter":
			// enter = next step
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m PostInstallModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("10 — POST-INSTALL") + "\n\n")

	for i, opt := range m.options {
		sel := "  "
		if i == m.cursor {
			sel = style.StyleSelected.Render("▶ ")
		}

		checked := opt.get()
		locked := opt.label == "Enable NetworkManager"

		cb := style.Checkbox(checked)
		if locked {
			cb = style.StyleGood.Render("[✓]") // locked green
		}

		b.WriteString(sel + cb + " " + style.StyleValue.Render(opt.label) + "\n")
		b.WriteString("       " + style.StyleMuted.Render(opt.desc) + "\n\n")
	}

	b.WriteString(style.HelpRow("↑↓/tab", "item", "space", "toggle", "ctrl+a", "advanced", "enter", "next", "esc", "back"))

	return b.String()
}
