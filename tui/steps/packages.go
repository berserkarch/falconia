package steps

import (
	"fmt"
	"strings"

	"falconia/config"
	"falconia/style"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pkgCategory struct {
	name     string
	packages []pkgEntry
}

type pkgEntry struct {
	name      string
	desc      string
	checked   bool
	isHeader  bool
	collapsed bool
	level     int // nesting level
}

// PackagesModel is a categorised multiselect for extra packages.
type PackagesModel struct {
	cfg        *config.InstallConfig
	advanced   bool
	categories []pkgCategory
	catIdx     int
	pkgIdx     int
	focused    int // 0: categories, 1: packages

	scrollOffset int
	pageSize     int
}

func defaultCategories() []pkgCategory {
	return []pkgCategory{
		{
			name: "Browsers & Internet",
			packages: []pkgEntry{
				{name: "Web Browsers", isHeader: true, level: 0},
				{name: "firefox", desc: "Mozilla Firefox web browser", level: 1},
				{name: "chromium", desc: "Chromium open-source browser", level: 1},
				{name: "Lightweight Browsers", isHeader: true, level: 1},
				{name: "falkon", desc: "KDE web browser", level: 2},
				{name: "midori", desc: "Lightweight web browser", level: 2},
				{name: "lynx", desc: "Text-based web browser", level: 2},
				{name: "Communication", isHeader: true, level: 0},
				{name: "discord", desc: "All-in-one voice and text chat", level: 1},
				{name: "telegram-desktop", desc: "Telegram Desktop client", level: 1},
				{name: "File Transfer", isHeader: true, level: 0},
				{name: "transmission-qt", desc: "BitTorrent client (Qt)", level: 1},
				{name: "qbittorrent", desc: "BitTorrent client (Qt6)", level: 1},
			},
		},
		{
			name: "Dev Tools",
			packages: []pkgEntry{
				{name: "Version Control", isHeader: true},
				{name: "git", desc: "Version control system", level: 1},
				{name: "github-cli", desc: "GitHub CLI tool", level: 1},
				{name: "Languages", isHeader: true},
				{name: "Python", isHeader: true, level: 1},
				{name: "python", desc: "Python 3 interpreter", level: 2},
				{name: "python-pip", desc: "Python package installer", level: 2},
				{name: "Go", isHeader: true, level: 1},
				{name: "go", desc: "Go programming language", level: 2},
				{name: "Rust", isHeader: true, level: 1},
				{name: "rustup", desc: "Rust toolchain manager", level: 2},
				{name: "Editors", isHeader: true},
				{name: "neovim", desc: "Extensible terminal editor", level: 1},
				{name: "emacs", desc: "Extensible, self-documenting editor", level: 1},
			},
		},
		{
			name: "Services",
			packages: []pkgEntry{
				{name: "System Daemons", isHeader: true},
				{name: "docker", desc: "Container orchestration daemon", level: 1},
				{name: "bluez", desc: "Bluetooth stack daemon", level: 1},
				{name: "cups", desc: "Common Unix Printing System", level: 1},
				{name: "Networking Services", isHeader: true},
				{name: "openssh", desc: "OpenSSH server daemon", level: 1},
				{name: "tailscale", desc: "Zero config VPN", level: 1},
				{name: "Databases", isHeader: true},
				{name: "mariadb", desc: "MariaDB SQL database server", level: 1},
				{name: "postgresql", desc: "PostgreSQL database server", level: 1},
				{name: "redis", desc: "Advanced key-value store", level: 1},
			},
		},
		{
			name: "Security",
			packages: []pkgEntry{
				{name: "Encryption", isHeader: true},
				{name: "gnupg", desc: "GNU Privacy Guard", level: 1},
				{name: "age", desc: "Simple, modern file encryption tool", level: 1},
				{name: "Password Managers", isHeader: true},
				{name: "keepassxc", desc: "Community-driven password manager", level: 1},
				{name: "pass", desc: "Standard Unix password manager", level: 1},
				{name: "Audit & Analysis", isHeader: true},
				{name: "nmap", desc: "Network mapper", level: 1},
				{name: "wireshark-qt", desc: "Network protocol analyzer", level: 1},
				{name: "lynis", desc: "Security auditing tool", level: 1},
			},
		},
		{
			name: "Desktop Base + Common Packages",
			packages: []pkgEntry{
				{name: "mesa-utils", desc: "mesa-utils", level: 1},
				{name: "xf86-input-libinput", level: 1},
				{name: "extra/xorg", level: 1},
				{name: "extra/xorg-xdpyinfo", level: 1},
				{name: "extra/xorg-server", level: 1},
				{name: "extra/xorg-xinit", level: 1},
				{name: "extra/xorg-xinput", level: 1},
				{name: "extra/xorg-xkill", level: 1},
				{name: "extra/xorg-xrandr", level: 1},
				{name: "Default X11 System", isHeader: true},
				{name: "gnupg", desc: "GNU Privacy Guard", level: 1},
				{name: "age", desc: "Simple, modern file encryption tool", level: 1},
				{name: "Password Managers", isHeader: true},
				{name: "keepassxc", desc: "Community-driven password manager", level: 1},
				{name: "pass", desc: "Standard Unix password manager", level: 1},
				{name: "Audit & Analysis", isHeader: true},
				{name: "nmap", desc: "Network mapper", level: 1},
				{name: "wireshark-qt", desc: "Network protocol analyzer", level: 1},
				{name: "lynis", desc: "Security auditing tool", level: 1},
			},
		},
	}
}

func NewPackages(cfg *config.InstallConfig, advanced bool) PackagesModel {
	cats := defaultCategories()

	// Pre-check packages already in cfg
	for ci := range cats {
		for pi := range cats[ci].packages {
			if cats[ci].packages[pi].isHeader {
				continue
			}
			for _, ep := range cfg.ExtraPackages {
				if ep == cats[ci].packages[pi].name {
					cats[ci].packages[pi].checked = true
				}
			}
		}
	}

	return PackagesModel{
		cfg:        cfg,
		advanced:   advanced,
		categories: cats,
		pageSize:   15,
		focused:    1, // Default to packages
	}
}

func (m PackagesModel) Init() tea.Cmd { return nil }

func (m PackagesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "up", "k":
			if m.focused == 0 {
				if m.catIdx > 0 {
					m.catIdx--
					m.pkgIdx = 0
					m.scrollOffset = 0
				}
			} else {
				m.moveCursor(-1)
				m.fixScroll()
			}

		case "down", "j":
			if m.focused == 0 {
				if m.catIdx < len(m.categories)-1 {
					m.catIdx++
					m.pkgIdx = 0
					m.scrollOffset = 0
				}
			} else {
				m.moveCursor(1)
				m.fixScroll()
			}

		case "tab":
			m.focused = (m.focused + 1) % 2
			return m, nil

		case "shift+tab":
			m.focused = (m.focused - 1 + 2) % 2
			return m, nil

		case "left", "h":
			if m.focused == 1 {
				pkg := m.categories[m.catIdx].packages[m.pkgIdx]
				if pkg.isHeader && !pkg.collapsed {
					m.categories[m.catIdx].packages[m.pkgIdx].collapsed = true
				} else {
					m.focused = 0
				}
			}

		case "right", "l":
			if m.focused == 0 {
				m.focused = 1
			} else {
				pkg := m.categories[m.catIdx].packages[m.pkgIdx]
				if pkg.isHeader && pkg.collapsed {
					m.categories[m.catIdx].packages[m.pkgIdx].collapsed = false
				}
			}

		case " ":
			if m.focused == 1 {
				pkg := &m.categories[m.catIdx].packages[m.pkgIdx]
				if pkg.isHeader {
					pkg.collapsed = !pkg.collapsed
				} else {
					pkg.checked = !pkg.checked
				}
			}

		case "enter":
			if m.focused == 0 {
				m.focused = 1
				return m, nil
			}
			m.Save()
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m *PackagesModel) moveCursor(delta int) {
	cat := m.categories[m.catIdx]
	newIdx := m.pkgIdx

	for {
		newIdx += delta
		if newIdx < 0 {
			// Wrap to end
			newIdx = len(cat.packages) - 1
		} else if newIdx >= len(cat.packages) {
			// Wrap to start
			newIdx = 0
		}

		if m.isRowVisible(newIdx) {
			m.pkgIdx = newIdx
			break
		}

		// If we've looped through everything and nothing is visible (shouldn't happen)
		if newIdx == m.pkgIdx {
			break
		}
	}
}

func (m PackagesModel) isRowVisible(idx int) bool {
	cat := m.categories[m.catIdx]
	pkg := cat.packages[idx]

	// Check all parent headers
	for i := idx - 1; i >= 0; i-- {
		parent := cat.packages[i]
		if parent.isHeader && parent.level < pkg.level {
			if parent.collapsed {
				return false
			}
		}
	}
	return true
}

func (m *PackagesModel) fixScroll() {
	if m.pkgIdx < m.scrollOffset {
		m.scrollOffset = m.pkgIdx
	}
	if m.pkgIdx >= m.scrollOffset+m.pageSize {
		m.scrollOffset = m.pkgIdx - m.pageSize + 1
	}
}

func (m PackagesModel) Save() {
	var pkgs []string
	for _, cat := range m.categories {
		for _, p := range cat.packages {
			if p.checked && !p.isHeader {
				pkgs = append(pkgs, p.name)
			}
		}
	}
	m.cfg.ExtraPackages = pkgs
}

func (m PackagesModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("08 — EXTRA PACKAGES") + "\n\n")

	// --- Category List (Left) ---
	var catList strings.Builder
	catWidth := 30
	for i, cat := range m.categories {
		sel := "  "
		var catName string
		if i == m.catIdx {
			if m.focused == 0 {
				sel = style.StyleSelected.Render("▶ ")
				catName = style.StyleSelected.Render(cat.name)
			} else {
				sel = style.StyleKey.Render("● ")
				catName = style.StyleKey.Render(cat.name)
			}
		} else {
			catName = style.StyleMuted.Render(cat.name)
		}

		line := sel + catName + "\n\n" // Extra spacing
		catList.WriteString(line)
	}

	catView := lipgloss.NewStyle().
		Width(catWidth).
		MarginRight(4).
		PaddingTop(1).
		Render(catList.String())

	// --- Package List (Right) ---
	var pkgList strings.Builder
	cat := m.categories[m.catIdx]

	// Top indicator
	if m.scrollOffset > 0 {
		pkgList.WriteString(style.StyleMuted.Render("      ↑ more...") + "\n\n")
	} else {
		pkgList.WriteString("\n\n")
	}

	endIdx := m.scrollOffset + m.pageSize
	if endIdx > len(cat.packages) {
		endIdx = len(cat.packages)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		if !m.isRowVisible(i) {
			continue
		}
		pkg := cat.packages[i]
		sel := "  "
		if i == m.pkgIdx && m.focused == 1 {
			sel = style.StyleSelected.Render("▶ ")
		}

		indent := strings.Repeat("  ", pkg.level)
		if pkg.isHeader {
			state := "▼"
			if pkg.collapsed {
				state = "▶"
			}
			pkgList.WriteString(sel + indent + style.StyleKey.Render(state+" "+pkg.name) + "\n\n")
		} else {
			pkgList.WriteString(sel + indent + style.Checkbox(pkg.checked) + " " + style.StyleValue.Render(pkg.name))
			if m.advanced {
				pkgList.WriteString("\n" + sel + indent + "    " + style.StyleMuted.Render(pkg.desc))
			}
			pkgList.WriteString("\n\n")
		}
	}

	// Bottom indicator
	if endIdx < len(cat.packages) {
		pkgList.WriteString(style.StyleMuted.Render("      ↓ more...") + "\n")
	} else {
		pkgList.WriteString("\n")
	}

	pkgView := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#444444")). // Vertical divider
		PaddingLeft(2).
		Render(pkgList.String())

	// Join side-by-side
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, catView, pkgView))

	// Summary
	total := 0
	for _, c := range m.categories {
		for _, p := range c.packages {
			if p.checked && !p.isHeader {
				total++
			}
		}
	}
	b.WriteString("\n\n" + style.StyleMuted.Render(fmt.Sprintf("  %d package(s) selected", total)) + "\n")

	b.WriteString("\n")
	b.WriteString(style.HelpRow(
		"↑↓", "move",
		"←→/tab", "focus/collapse",
		"space", "toggle/collapse",
		"ctrl+a", "advanced",
		"enter", "done",
		"esc", "back",
	))
	return b.String()
}
