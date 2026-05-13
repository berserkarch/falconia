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

type PostInstallModel struct {
	cfg          *config.InstallConfig
	options      []postOption
	cursor       int
	scrollOffset int
	pageSize     int
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
		{
			"Enable Docker",
			"Start docker.service on boot",
			func() bool { return cfg.ExtraServices["docker"] },
			func(v bool) { cfg.ExtraServices["docker"] = v },
		},
		{
			"Enable Tailscale",
			"Start tailscale.service on boot",
			func() bool { return cfg.ExtraServices["tailscale"] },
			func(v bool) { cfg.ExtraServices["tailscale"] = v },
		},
		{
			"Enable Avahi (mDNS)",
			"Start avahi-daemon.service on boot",
			func() bool { return cfg.ExtraServices["avahi"] },
			func(v bool) { cfg.ExtraServices["avahi"] = v },
		},
		{
			"Enable TLP (Power management)",
			"Start tlp.service on boot",
			func() bool { return cfg.ExtraServices["tlp"] },
			func(v bool) { cfg.ExtraServices["tlp"] = v },
		},
		{
			"Enable UFW (Firewall)",
			"Start ufw.service on boot",
			func() bool { return cfg.ExtraServices["ufw"] },
			func(v bool) { cfg.ExtraServices["ufw"] = v },
		},
		{
			"Enable Flatpak Support",
			"Install flatpak and configure repositories",
			func() bool { return cfg.ExtraServices["flatpak"] },
			func(v bool) { cfg.ExtraServices["flatpak"] = v },
		},
		{
			"Enable Snap Support",
			"Install snapd and start snapd.socket",
			func() bool { return cfg.ExtraServices["snap"] },
			func(v bool) { cfg.ExtraServices["snap"] = v },
		},
		{
			"Enable Zram",
			"Configure zram-generator for swap",
			func() bool { return cfg.ExtraServices["zram"] },
			func(v bool) { cfg.ExtraServices["zram"] = v },
		},
		{
			"Enable Periodic Trim",
			"Start fstrim.timer for SSD health",
			func() bool { return cfg.ExtraServices["trim"] },
			func(v bool) { cfg.ExtraServices["trim"] = v },
		},
		{
			"Enable Microcode Updates",
			"Install microcode for CPU security and stability",
			func() bool { return cfg.ExtraServices["microcode"] },
			func(v bool) { cfg.ExtraServices["microcode"] = v },
		},
	}
	// Set some defaults for the new ones
	if _, ok := cfg.ExtraServices["microcode"]; !ok {
		cfg.ExtraServices["microcode"] = true
	}

	return PostInstallModel{
		cfg:      cfg,
		options:  opts,
		pageSize: 8, // Show 8 items at a time
	}
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
			m.fixScroll()
		case "down", "j", "tab":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.fixScroll()
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

func (m *PostInstallModel) fixScroll() {
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+m.pageSize {
		m.scrollOffset = m.cursor - m.pageSize + 1
	}
}

func (m PostInstallModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("10 — POST-INSTALL") + "\n\n")

	// Top indicator
	if m.scrollOffset > 0 {
		b.WriteString(style.StyleMuted.Render("    ↑ more...") + "\n\n")
	} else {
		b.WriteString("\n\n")
	}

	endIdx := m.scrollOffset + m.pageSize
	if endIdx > len(m.options) {
		endIdx = len(m.options)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		opt := m.options[i]
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

	// Bottom indicator
	if endIdx < len(m.options) {
		b.WriteString(style.StyleMuted.Render("    ↓ more...") + "\n\n")
	} else {
		b.WriteString("\n\n")
	}

	b.WriteString(style.HelpRow("↑↓/tab", "item", "space", "toggle", "ctrl+a", "advanced", "enter", "next", "esc", "back"))

	return b.String()
}
