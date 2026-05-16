package steps

import (
	"falconia/config"
	"falconia/style"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ── Generic radio-list step ──────────────────────────────────────────────────

type radioOption struct {
	value string
	label string
	desc  string
}

type radioModel struct {
	cfg     *config.InstallConfig
	title   string
	options []radioOption
	cursor  int
	setter  func(string)
	getter  func() string
}

func (m radioModel) Init() tea.Cmd { return nil }

func (m radioModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter", " ":
			m.setter(m.options[m.cursor].value)
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m radioModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render(m.title) + "\n\n")
	current := m.getter()
	for i, opt := range m.options {
		sel := "  "
		if i == m.cursor {
			sel = style.StyleSelected.Render("▶ ")
		}
		mark := style.StyleMuted.Render("○ ")
		if opt.value == current {
			mark = style.StyleGood.Render("● ")
		}
		b.WriteString(sel + mark + style.StyleValue.Render(opt.label) + "\n")
		if opt.desc != "" {
			b.WriteString("      " + style.StyleMuted.Render(opt.desc) + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(style.HelpRow("↑↓", "select", "ctrl+a", "advanced", "enter", "confirm", "esc", "back"))
	return b.String()
}

// ── Step 06: Kernel ──────────────────────────────────────────────────────────

func NewKernel(cfg *config.InstallConfig) radioModel {
	return radioModel{
		cfg:   cfg,
		title: "06 — KERNEL",
		options: []radioOption{
			{"linux", "linux", "Latest stable kernel (recommended)"},
			{"linux-lts", "linux-lts", "Long-term support kernel — older but very stable"},
			{"linux-zen", "linux-zen", "Optimized for desktop / gaming — better latency"},
			{"linux-hardened", "linux-hardened", "Security-focused with extra patches — more restrictive"},
		},
		setter: func(v string) { cfg.Kernel = v },
		getter: func() string { return cfg.Kernel },
	}
}

// ── Step 07: Desktop Environment ────────────────────────────────────────────

type desktopOption struct {
	value  string
	label  string
	desc   string
	sizeMB int
}

// DesktopModel is a specialised radio for DE selection that shows size estimates.
type DesktopModel struct {
	cfg     *config.InstallConfig
	options []desktopOption
	cursor  int
}

func NewDesktop(cfg *config.InstallConfig) DesktopModel {
	options := []desktopOption{
		{"none", "None (minimal)", "No desktop — TTY only", 0},
		{"gnome", "GNOME", "Full GNOME desktop + GDM", 800},
		{"kde", "KDE Plasma", "Full Plasma desktop + SDDM", 700},
		{"xfce", "XFCE", "Lightweight desktop + LightDM", 250},
		{"cinnamon", "Cinnamon", "Traditional desktop + LightDM", 400},
		{"hyprland", "Hyprland", "Wayland compositor + SDDM + waybar", 180},
		{"i3", "i3", "Tiling window manager + SDDM + xorg", 120},
		{"openbox", "Openbox", "Stacking window manager + SDDM", 80},
	}
	cursor := 0
	for i, o := range options {
		if o.value == cfg.DesktopEnv {
			cursor = i
		}
	}
	return DesktopModel{cfg: cfg, options: options, cursor: cursor}
}

func (m DesktopModel) Init() tea.Cmd { return nil }

func (m DesktopModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter", " ":
			opt := m.options[m.cursor]
			m.cfg.DesktopEnv = opt.value
			m.cfg.DisplayManager = config.DEDisplayManager(opt.value)
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m DesktopModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("07 — DESKTOP ENVIRONMENT") + "\n\n")
	for i, opt := range m.options {
		sel := "  "
		if i == m.cursor {
			sel = style.StyleSelected.Render("▶ ")
		}
		mark := style.StyleMuted.Render("○ ")
		if opt.value == m.cfg.DesktopEnv {
			mark = style.StyleGood.Render("● ")
		}
		size := ""
		if opt.sizeMB > 0 {
			size = style.StyleMuted.Render(fmt.Sprintf("  ~%d MB", opt.sizeMB))
		}
		b.WriteString(sel + mark + style.StyleValue.Render(opt.label) + size + "\n")
		b.WriteString("      " + style.StyleMuted.Render(opt.desc) + "\n\n")
	}
	b.WriteString(style.HelpRow("↑↓", "select", "ctrl+a", "advanced", "enter", "confirm", "esc", "back"))
	return b.String()
}

// ── Step 09: Bootloader ──────────────────────────────────────────────────────

// BootloaderModel is a radio that hides systemd-boot on BIOS systems.
type BootloaderModel struct {
	cfg     *config.InstallConfig
	options []radioOption
	cursor  int
}

func NewBootloader(cfg *config.InstallConfig) BootloaderModel {
	opts := []radioOption{
		{"grub", "GRUB", "Works on both UEFI and legacy BIOS — most compatible"},
	}
	if cfg.Firmware == "uefi" {
		opts = append(opts, radioOption{
			"systemd-boot", "systemd-boot", "Lightweight UEFI-only bootloader — minimal and fast",
		})
	}
	cursor := 0
	for i, o := range opts {
		if o.value == cfg.Bootloader {
			cursor = i
		}
	}
	return BootloaderModel{cfg: cfg, options: opts, cursor: cursor}
}

func (m BootloaderModel) Init() tea.Cmd { return nil }

func (m BootloaderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter", " ":
			m.cfg.Bootloader = m.options[m.cursor].value
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m BootloaderModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("09 — BOOTLOADER") + "\n\n")

	if m.cfg.Firmware == "bios" {
		b.WriteString(style.StyleMuted.Render("  Legacy BIOS detected — GRUB is the only option.\n\n"))
	}

	for i, opt := range m.options {
		sel := "  "
		if i == m.cursor {
			sel = style.StyleSelected.Render("▶ ")
		}
		mark := style.StyleMuted.Render("○ ")
		if opt.value == m.cfg.Bootloader {
			mark = style.StyleGood.Render("● ")
		}
		b.WriteString(sel + mark + style.StyleValue.Render(opt.label) + "\n")
		b.WriteString("      " + style.StyleMuted.Render(opt.desc) + "\n\n")
	}
	b.WriteString(style.HelpRow("↑↓", "select", "ctrl+a", "advanced", "enter", "confirm", "esc", "back"))
	return b.String()
}
