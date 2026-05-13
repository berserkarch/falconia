package steps

import (
	"falconia/config"
	"falconia/installer"
	"falconia/style"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type localeField int

const (
	localeFieldTZ localeField = iota
	localeFieldLocale
	localeFieldKeymap
)

// LocaleModel handles timezone, locale, and keymap selection.
type LocaleModel struct {
	cfg    *config.InstallConfig
	cursor localeField

	tzInput     textinput.Model
	localeInput textinput.Model
	keymapInput textinput.Model

	allZones      []string
	filteredZones []string
	tzCursor      int

	allLocales      []string
	filteredLocales []string
	locCursor       int

	err string
}

func NewLocale(cfg *config.InstallConfig) LocaleModel {
	tz := textinput.New()
	tz.Placeholder = "Type to search timezone... (e.g. London)"
	tz.Width = 50
	tz.SetValue(cfg.Timezone)
	tz.Focus()

	loc := textinput.New()
	loc.Placeholder = "Type to search locale... (e.g. en_US)"
	loc.Width = 50
	loc.SetValue(cfg.Locale)

	km := textinput.New()
	km.Placeholder = "e.g. us"
	km.Width = 30
	km.SetValue(cfg.Keymap)

	zones := installer.ListTimezones()
	locales := installer.ListLocales()

	m := LocaleModel{
		cfg:         cfg,
		tzInput:     tz,
		localeInput: loc,
		keymapInput: km,
		allZones:    zones,
		allLocales:  locales,
	}
	m.updateFilteredZones()
	m.updateFilteredLocales()
	return m
}

func (m LocaleModel) Init() tea.Cmd { return m.tzInput.Focus() }

func (m LocaleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "up":
			if m.cursor == localeFieldTZ && len(m.filteredZones) > 0 {
				if m.tzCursor > 0 {
					m.tzCursor--
				}
				return m, nil
			}
			if m.cursor == localeFieldLocale && len(m.filteredLocales) > 0 {
				if m.locCursor > 0 {
					m.locCursor--
				}
				return m, nil
			}
			fallthrough
		case "k", "shift+tab":
			if msg.String() == "k" {
				break
			}
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = localeFieldKeymap
			}
			m.updateFocus()
			return m, nil
		case "down":
			if m.cursor == localeFieldTZ && len(m.filteredZones) > 0 {
				if m.tzCursor < len(m.filteredZones)-1 && m.tzCursor < 4 { // max 5 suggestions
					m.tzCursor++
				}
				return m, nil
			}
			if m.cursor == localeFieldLocale && len(m.filteredLocales) > 0 {
				if m.locCursor < len(m.filteredLocales)-1 && m.locCursor < 4 { // max 5 suggestions
					m.locCursor++
				}
				return m, nil
			}
			fallthrough
		case "j", "tab":
			if msg.String() == "j" {
				break
			}
			if m.cursor < localeFieldKeymap {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.updateFocus()
			return m, nil
		case "enter":
			if m.cursor == localeFieldTZ && len(m.filteredZones) > 0 {
				m.tzInput.SetValue(m.filteredZones[m.tzCursor])
				m.filteredZones = nil
				m.cursor++
				m.updateFocus()
				return m, nil
			}
			if m.cursor == localeFieldLocale && len(m.filteredLocales) > 0 {
				m.localeInput.SetValue(m.filteredLocales[m.locCursor])
				m.filteredLocales = nil
				m.cursor++
				m.updateFocus()
				return m, nil
			}
			if m.cursor < localeFieldKeymap {
				m.cursor++
				m.updateFocus()
				return m, nil
			}
			if err := m.validate(); err != "" {
				m.err = err
				return m, nil
			}
			m.Save()
			return m, EmitDone()
		}
	}

	var cmd tea.Cmd
	oldTZ := m.tzInput.Value()
	oldLoc := m.localeInput.Value()
	switch m.cursor {
	case localeFieldTZ:
		m.tzInput, cmd = m.tzInput.Update(msg)
		if m.tzInput.Value() != oldTZ {
			m.updateFilteredZones()
		}
	case localeFieldLocale:
		m.localeInput, cmd = m.localeInput.Update(msg)
		if m.localeInput.Value() != oldLoc {
			m.updateFilteredLocales()
		}
	case localeFieldKeymap:
		m.keymapInput, cmd = m.keymapInput.Update(msg)
	}
	return m, cmd
}

func (m *LocaleModel) updateFilteredZones() {
	val := strings.ToLower(m.tzInput.Value())
	if val == "" {
		m.filteredZones = nil
		return
	}

	var filtered []string
	for _, z := range m.allZones {
		if strings.Contains(strings.ToLower(z), val) {
			filtered = append(filtered, z)
			if len(filtered) >= 5 {
				break
			}
		}
	}
	m.filteredZones = filtered
	m.tzCursor = 0
}

func (m *LocaleModel) updateFilteredLocales() {
	val := strings.ToLower(m.localeInput.Value())
	if val == "" {
		m.filteredLocales = nil
		return
	}

	var filtered []string
	for _, l := range m.allLocales {
		if strings.Contains(strings.ToLower(l), val) {
			filtered = append(filtered, l)
			if len(filtered) >= 5 {
				break
			}
		}
	}
	m.filteredLocales = filtered
	m.locCursor = 0
}

func (m *LocaleModel) updateFocus() {
	m.tzInput.Blur()
	m.localeInput.Blur()
	m.keymapInput.Blur()
	switch m.cursor {
	case localeFieldTZ:
		m.tzInput.Focus()
		m.updateFilteredZones()
	case localeFieldLocale:
		m.localeInput.Focus()
		m.updateFilteredLocales()
	case localeFieldKeymap:
		m.keymapInput.Focus()
	}
}

func (m LocaleModel) validate() string {
	if strings.TrimSpace(m.tzInput.Value()) == "" {
		return "Timezone is required (e.g. America/New_York)"
	}
	// Basic check if it's a valid system timezone
	foundTZ := false
	tzVal := strings.TrimSpace(m.tzInput.Value())
	for _, z := range m.allZones {
		if z == tzVal {
			foundTZ = true
			break
		}
	}
	if !foundTZ {
		return "Invalid timezone. Please select from suggestions or use Region/City format."
	}

	if strings.TrimSpace(m.localeInput.Value()) == "" {
		return "Locale is required (e.g. en_US.UTF-8)"
	}
	// Basic check if it's a valid system locale
	foundLoc := false
	locVal := strings.TrimSpace(m.localeInput.Value())
	for _, l := range m.allLocales {
		if l == locVal {
			foundLoc = true
			break
		}
	}
	if !foundLoc {
		return "Invalid locale. Please select from suggestions or use xx_XX.UTF-8 format."
	}

	if strings.TrimSpace(m.keymapInput.Value()) == "" {
		return "Keymap is required (e.g. us)"
	}
	return ""
}

func (m LocaleModel) Save() {
	m.cfg.Timezone = strings.TrimSpace(m.tzInput.Value())
	m.cfg.Locale = strings.TrimSpace(m.localeInput.Value())
	m.cfg.Keymap = strings.TrimSpace(m.keymapInput.Value())
}

func (m LocaleModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("03 — LOCALE") + "\n\n")

	// Timezone with autocomplete
	cursorTZ := "  "
	if m.cursor == localeFieldTZ {
		cursorTZ = style.StyleSelected.Render("▶ ")
	}
	b.WriteString(cursorTZ + style.StyleKey.Render(fmt.Sprintf("%-12s", "Timezone")) + m.tzInput.View() + "\n")
	
	// TZ Autocomplete suggestions
	if m.cursor == localeFieldTZ && len(m.filteredZones) > 0 {
		for i, z := range m.filteredZones {
			if i == m.tzCursor {
				b.WriteString("              " + style.StyleSelected.Render("> "+z) + "\n")
			} else {
				b.WriteString("                " + style.StyleMuted.Render(z) + "\n")
			}
		}
	} else {
		b.WriteString(style.StyleMuted.Render("              zoneinfo path, e.g. Europe/Berlin") + "\n")
	}
	b.WriteString("\n")

	// Locale with autocomplete
	cursorLoc := "  "
	if m.cursor == localeFieldLocale {
		cursorLoc = style.StyleSelected.Render("▶ ")
	}
	b.WriteString(cursorLoc + style.StyleKey.Render(fmt.Sprintf("%-12s", "Locale")) + m.localeInput.View() + "\n")

	// Locale Autocomplete suggestions
	if m.cursor == localeFieldLocale && len(m.filteredLocales) > 0 {
		for i, l := range m.filteredLocales {
			if i == m.locCursor {
				b.WriteString("              " + style.StyleSelected.Render("> "+l) + "\n")
			} else {
				b.WriteString("                " + style.StyleMuted.Render(l) + "\n")
			}
		}
	} else {
		b.WriteString(style.StyleMuted.Render("              e.g. en_US.UTF-8  ja_JP.UTF-8") + "\n")
	}
	b.WriteString("\n")

	// Keymap (no autocomplete yet)
	cursorKM := "  "
	if m.cursor == localeFieldKeymap {
		cursorKM = style.StyleSelected.Render("▶ ")
	}
	b.WriteString(cursorKM + style.StyleKey.Render(fmt.Sprintf("%-12s", "Keymap")) + m.keymapInput.View() + "\n")
	b.WriteString(style.StyleMuted.Render("              loadkeys name, e.g. us  uk  de") + "\n\n")

	if m.err != "" {
		b.WriteString(style.StyleError.Render("  "+m.err) + "\n\n")
	}

	b.WriteString(style.HelpRow("↑↓/tab", "field/select", "enter", "next/confirm", "esc", "back"))
	return b.String()
}
