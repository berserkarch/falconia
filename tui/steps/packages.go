package steps

import (
	"fmt"
	"strings"

	"falconia/config"
	"falconia/data"
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

// defaultCategories converts data.ExtraCategories into the TUI's internal
// representation. Static package data lives in data/extras.go; runtime state
// (checked, collapsed) stays here in the model.
func defaultCategories() []pkgCategory {
	cats := make([]pkgCategory, len(data.ExtraCategories))
	for i, c := range data.ExtraCategories {
		entries := make([]pkgEntry, len(c.Entries))
		for j, e := range c.Entries {
			entries[j] = pkgEntry{
				name:     e.Name,
				desc:     e.Desc,
				isHeader: e.IsHeader,
				level:    e.Level,
			}
		}
		cats[i] = pkgCategory{name: c.Name, packages: entries}
	}
	return cats
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
