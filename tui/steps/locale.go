package steps

import (
	"falconia/config"
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

	err string
}

func NewLocale(cfg *config.InstallConfig) LocaleModel {
	tz := textinput.New()
	tz.Placeholder = "Region/City  e.g. Asia/Kolkata"
	tz.Width = 50
	tz.SetValue(cfg.Timezone)
	tz.Focus()

	loc := textinput.New()
	loc.Placeholder = "e.g. en_US.UTF-8"
	loc.Width = 50
	loc.SetValue(cfg.Locale)

	km := textinput.New()
	km.Placeholder = "e.g. us"
	km.Width = 30
	km.SetValue(cfg.Keymap)

	return LocaleModel{
		cfg:         cfg,
		tzInput:     tz,
		localeInput: loc,
		keymapInput: km,
	}
}

func (m LocaleModel) Init() tea.Cmd { return m.tzInput.Focus() }

func (m LocaleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "up", "k", "shift+tab":
			if msg.String() == "k" {
				// all fields are text inputs
				break
			}
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = localeFieldKeymap
			}
			m.updateFocus()
		case "down", "j", "tab":
			if msg.String() == "j" {
				// all fields are text inputs
				break
			}
			if m.cursor < localeFieldKeymap {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.updateFocus()
		case "enter":
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
	switch m.cursor {
	case localeFieldTZ:
		m.tzInput, cmd = m.tzInput.Update(msg)
	case localeFieldLocale:
		m.localeInput, cmd = m.localeInput.Update(msg)
	case localeFieldKeymap:
		m.keymapInput, cmd = m.keymapInput.Update(msg)
	}
	return m, cmd
}

func (m *LocaleModel) updateFocus() {
	m.tzInput.Blur()
	m.localeInput.Blur()
	m.keymapInput.Blur()
	switch m.cursor {
	case localeFieldTZ:
		m.tzInput.Focus()
	case localeFieldLocale:
		m.localeInput.Focus()
	case localeFieldKeymap:
		m.keymapInput.Focus()
	}
}

func (m LocaleModel) validate() string {
	if strings.TrimSpace(m.tzInput.Value()) == "" {
		return "Timezone is required (e.g. America/New_York)"
	}
	if strings.TrimSpace(m.localeInput.Value()) == "" {
		return "Locale is required (e.g. en_US.UTF-8)"
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

	renderRow := func(f localeField, label, desc string, input textinput.Model) {
		cursor := "  "
		if m.cursor == f {
			cursor = style.StyleSelected.Render("▶ ")
		}
		labelStr := style.StyleKey.Render(fmt.Sprintf("%-12s", label))
		b.WriteString(cursor + labelStr + input.View() + "\n")
		b.WriteString(style.StyleMuted.Render("              "+desc) + "\n\n")
	}

	renderRow(localeFieldTZ, "Timezone", "zoneinfo path, e.g. Europe/Berlin", m.tzInput)
	renderRow(localeFieldLocale, "Locale", "e.g. en_US.UTF-8  ja_JP.UTF-8", m.localeInput)
	renderRow(localeFieldKeymap, "Keymap", "loadkeys name, e.g. us  uk  de", m.keymapInput)

	if m.err != "" {
		b.WriteString(style.StyleError.Render("  "+m.err) + "\n\n")
	}

	b.WriteString(style.HelpRow("↑↓/tab", "field", "ctrl+a", "advanced", "enter", "next", "esc", "back"))
	return b.String()
}
