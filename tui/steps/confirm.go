package steps

import (
	"falconia/config"
	"falconia/style"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmModel shows a full read-only summary and two buttons.
type ConfirmModel struct {
	cfg      *config.InstallConfig
	cursor   int  // 0 = install, 1 = go back
	expanded bool // toggle package list
}

func NewConfirm(cfg *config.InstallConfig) ConfirmModel {
	return ConfirmModel{cfg: cfg}
}

func (m ConfirmModel) Init() tea.Cmd { return nil }

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "left", "h":
			m.cursor = 0
		case "right", "l":
			m.cursor = 1
		case "tab":
			m.cursor = 1 - m.cursor
		case "v":
			m.expanded = !m.expanded
		case "enter":
			if m.cursor == 0 {
				return m, EmitStartInstall()
			}
			return m, EmitBack()
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	cfg := m.cfg
	var b strings.Builder

	b.WriteString(style.StyleStepHeader.Render("CONFIRM INSTALLATION") + "\n\n")

	row := func(key, val string) {
		k := fmt.Sprintf("%-18s", key)
		lines := strings.Split(val, "\n")
		b.WriteString(style.StyleKey.Render(k) + " " + style.StyleValue.Render(lines[0]) + "\n")
		for i := 1; i < len(lines); i++ {
			b.WriteString(strings.Repeat(" ", 19) + style.StyleValue.Render(lines[i]) + "\n")
		}
	}

	b.WriteString(style.StyleSubtitle.Render("System") + "\n")
	row("Firmware", cfg.Firmware)
	row("Disk", fmt.Sprintf("%s — %s", cfg.Disk, cfg.DiskModel))
	row("Partitioning", cfg.PartitionScheme)
	row("Filesystem", cfg.Filesystem)

	if cfg.EncryptDisk {
		row("Encryption", "LUKS (Enabled)")
	} else {
		row("Encryption", "Disabled")
	}

	if cfg.SwapSize > 0 {
		row("Swap", fmt.Sprintf("%d MiB", cfg.SwapSize))
	} else {
		row("Swap", "none")
	}
	b.WriteString("\n")

	b.WriteString(style.StyleSubtitle.Render("Network") + "\n")
	switch cfg.NetworkMode {
	case "wifi":
		row("Network", "WiFi: "+cfg.WifiSSID)
	case "ethernet":
		row("Network", "Ethernet")
	default:
		row("Network", "Skipped")
	}
	b.WriteString("\n")

	b.WriteString(style.StyleSubtitle.Render("Locale") + "\n")
	row("Timezone", cfg.Timezone)
	row("Locale", cfg.Locale)
	row("Keymap", cfg.Keymap)
	b.WriteString("\n")

	b.WriteString(style.StyleSubtitle.Render("Identity") + "\n")
	row("Hostname", cfg.Hostname)
	for _, u := range cfg.Users {
		row("User", fmt.Sprintf("%s  shell=%s  groups=%s",
			u.Username, u.Shell, strings.Join(u.Groups, ",")))
	}
	b.WriteString("\n")

	b.WriteString(style.StyleSubtitle.Render("Software") + "\n")
	row("Kernel", cfg.Kernel)
	row("Desktop", orDash(cfg.DesktopEnv, "none"))
	row("Bootloader", cfg.Bootloader)
	if len(cfg.ExtraPackages) > 0 {
		row("Packages", m.formatFoldableList(cfg.ExtraPackages))
	}
	b.WriteString("\n")

	if cfg.RankMirrors {
		b.WriteString(style.StyleSubtitle.Render("Post-install") + "\n")
		row("Mirrors", "rank by speed before install")
		b.WriteString("\n")
	}

	b.WriteString(style.StyleDanger.Render("⚠  This will permanently erase "+cfg.Disk) + "\n\n")

	// Buttons
	var installBtn, backBtn string
	if m.cursor == 0 {
		installBtn = style.StyleButtonActive.Render("INSTALL")
		backBtn = style.StyleButtonInactive.Render("GO BACK")
	} else {
		installBtn = style.StyleButtonInactive.Render("INSTALL")
		backBtn = style.StyleButtonDanger.Render("GO BACK")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Top, installBtn, "    ", backBtn)
	b.WriteString(buttons + "\n\n")

	b.WriteString(style.HelpRow("←→/tab", "button", "v", "toggle view", "enter", "confirm", "esc", "back"))
	return b.String()
}

func (m ConfirmModel) formatFoldableList(items []string) string {
	str := strings.Join(items, "  ")
	if !m.expanded {
		wrapped := lipgloss.NewStyle().Width(80).Render(str)
		lines := strings.Split(wrapped, "\n")
		if len(lines) > 2 {
			str = strings.Join(lines[:2], "\n")
			if len(str) > 3 {
				str = str[:len(str)-3] + "..."
			} else {
				str += "..."
			}
			str += style.StyleMuted.Render(" (v=show all)")
		}
	} else {
		str = lipgloss.NewStyle().Width(80).Render(str)
		// Only show (v=hide) if it actually wrapped/is long
		if len(items) > 5 || len(strings.Join(items, "  ")) > 60 {
			str += style.StyleMuted.Render(" (v=hide)")
		}
	}
	return str
}

func orDash(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
