package steps

import (
	"fmt"
	"strings"

	"falconia/config"
	"falconia/style"

	tea "github.com/charmbracelet/bubbletea"
)

type pkgCategory struct {
	name     string
	packages []pkgEntry
}

type pkgEntry struct {
	name    string
	desc    string
	checked bool
}

// PackagesModel is a categorised multiselect for extra packages.
type PackagesModel struct {
	cfg        *config.InstallConfig
	advanced   bool
	categories []pkgCategory
	catIdx     int
	pkgIdx     int
}

func defaultCategories() []pkgCategory {
	return []pkgCategory{
		{
			name: "Audio",
			packages: []pkgEntry{
				{"pipewire", "Modern audio server", false},
				{"wireplumber", "PipeWire session manager", false},
				{"pipewire-pulse", "PulseAudio compatibility", false},
				{"pipewire-alsa", "ALSA compatibility", false},
			},
		},
		{
			name: "Networking",
			packages: []pkgEntry{
				{"iwd", "iNet Wireless Daemon (WiFi)", false},
				{"openssh", "SSH client and server", false},
				{"wget", "HTTP downloader", false},
				{"curl", "URL transfer tool", false},
			},
		},
		{
			name: "Fonts",
			packages: []pkgEntry{
				{"noto-fonts", "Google Noto font family", false},
				{"noto-fonts-emoji", "Noto emoji font", false},
				{"ttf-jetbrains-mono", "JetBrains Mono programmer font", false},
				{"ttf-firacode-nerd", "FiraCode Nerd Font", false},
			},
		},
		{
			name: "Dev Tools",
			packages: []pkgEntry{
				{"git", "Version control", false},
				{"docker", "Container runtime", false},
				{"neovim", "Terminal text editor", false},
				{"python", "Python 3 interpreter", false},
				{"rustup", "Rust toolchain manager", false},
				{"go", "Go programming language", false},
			},
		},
		{
			name: "Shell & Terminal",
			packages: []pkgEntry{
				{"zsh", "Z shell", false},
				{"fish", "Friendly interactive shell", false},
				{"bash-completion", "Bash tab completions", false},
				{"tmux", "Terminal multiplexer", false},
				{"alacritty", "GPU-accelerated terminal", false},
				{"kitty", "Feature-rich terminal", false},
			},
		},
		{
			name: "Utilities",
			packages: []pkgEntry{
				{"htop", "Interactive process viewer", false},
				{"btop", "Resource monitor", false},
				{"bat", "cat with syntax highlighting", false},
				{"ripgrep", "Fast recursive grep", false},
				{"fzf", "Fuzzy finder", false},
				{"fd", "Fast file finder", false},
				{"eza", "Modern ls replacement", false},
				{"unzip", "Zip extraction", false},
				{"p7zip", "7-Zip archiver", false},
			},
		},
		{
			name: "Bluetooth",
			packages: []pkgEntry{
				{"bluez", "Bluetooth protocol stack", false},
				{"bluez-utils", "Bluetooth CLI tools", false},
			},
		},
		{
			name: "Printing",
			packages: []pkgEntry{
				{"cups", "CUPS printing system", false},
				{"cups-pdf", "Print to PDF", false},
			},
		},
	}
}

func NewPackages(cfg *config.InstallConfig, advanced bool) PackagesModel {
	cats := defaultCategories()

	// Pre-check packages already in cfg
	for ci := range cats {
		for pi := range cats[ci].packages {
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
	}
}

func (m PackagesModel) Init() tea.Cmd { return nil }

func (m PackagesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "up", "k", "shift+tab":
			if m.pkgIdx > 0 {
				m.pkgIdx--
			} else if m.catIdx > 0 {
				m.catIdx--
				m.pkgIdx = len(m.categories[m.catIdx].packages) - 1
			} else {
				// wrap to last category, last package
				m.catIdx = len(m.categories) - 1
				m.pkgIdx = len(m.categories[m.catIdx].packages) - 1
			}

		case "down", "j", "tab":
			cat := m.categories[m.catIdx]
			if m.pkgIdx < len(cat.packages)-1 {
				m.pkgIdx++
			} else if m.catIdx < len(m.categories)-1 {
				m.catIdx++
				m.pkgIdx = 0
			} else {
				// wrap to first category, first package
				m.catIdx = 0
				m.pkgIdx = 0
			}

		case "left", "h":
			// previous category
			if m.catIdx > 0 {
				m.catIdx--
				m.pkgIdx = 0
			}

		case "right", "l":
			// next category
			if m.catIdx < len(m.categories)-1 {
				m.catIdx++
				m.pkgIdx = 0
			}

		case " ":
			m.categories[m.catIdx].packages[m.pkgIdx].checked = !m.categories[m.catIdx].packages[m.pkgIdx].checked

		case "enter":
			m.Save()
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m PackagesModel) Save() {
	var pkgs []string
	for _, cat := range m.categories {
		for _, p := range cat.packages {
			if p.checked {
				pkgs = append(pkgs, p.name)
			}
		}
	}
	m.cfg.ExtraPackages = pkgs
}

func (m PackagesModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("08 — EXTRA PACKAGES") + "\n\n")

	// Category tabs
	for i, cat := range m.categories {
		if i == m.catIdx {
			b.WriteString(style.StyleSelected.Render("["+cat.name+"]") + " ")
		} else {
			b.WriteString(style.StyleMuted.Render(" "+cat.name+" ") + " ")
		}
	}
	b.WriteString("\n\n")

	// Package list for active category
	cat := m.categories[m.catIdx]
	for pi, pkg := range cat.packages {
		sel := "  "
		if pi == m.pkgIdx {
			sel = style.StyleSelected.Render("▶ ")
		}
		b.WriteString(sel + style.Checkbox(pkg.checked) + " " + style.StyleValue.Render(pkg.name))
		if m.advanced {
			b.WriteString("  " + style.StyleMuted.Render(pkg.desc))
		}
		b.WriteString("\n")
	}

	// Summary
	total := 0
	for _, c := range m.categories {
		for _, p := range c.packages {
			if p.checked {
				total++
			}
		}
	}
	b.WriteString("\n" + style.StyleMuted.Render("  ") + style.StyleGood.Render(strings.Repeat("", total)))
	b.WriteString(style.StyleMuted.Render(fmt.Sprintf("  %d package(s) selected", total)) + "\n")

	b.WriteString("\n")
	b.WriteString(style.HelpRow(
		"↑↓/tab", "item",
		"←→", "category",
		"space", "toggle",
		"ctrl+a", "advanced",
		"enter", "done",
		"esc", "back",
	))
	return b.String()
}
