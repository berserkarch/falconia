package steps

import (
	"strings"

	"falconia/config"
	"falconia/style"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type netField int

const (
	netMode netField = iota
	netSSID
	netPass
)

// NetworkModel handles WiFi / ethernet / skip selection.
type NetworkModel struct {
	cfg     *config.InstallConfig
	modeIdx int // 0=ethernet, 1=wifi, 2=skip
	cursor  netField
	ssid    textinput.Model
	pass    textinput.Model
	err     string
}

var netModes = []string{"ethernet", "wifi", "skip"}

func NewNetwork(cfg *config.InstallConfig) NetworkModel {
	ssid := textinput.New()
	ssid.Placeholder = "MyNetwork"
	ssid.Width = 45
	ssid.SetValue(cfg.WifiSSID)

	pass := textinput.New()
	pass.Placeholder = "password"
	pass.EchoMode = textinput.EchoPassword
	pass.EchoCharacter = '•'
	pass.Width = 45
	pass.SetValue(cfg.WifiPass)

	modeIdx := 0
	for i, m := range netModes {
		if m == cfg.NetworkMode {
			modeIdx = i
		}
	}

	return NetworkModel{
		cfg:     cfg,
		modeIdx: modeIdx,
		ssid:    ssid,
		pass:    pass,
	}
}

func (m NetworkModel) Init() tea.Cmd { return nil }

func (m NetworkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "tab", "down", "j":
			if msg.String() == "j" && (m.cursor == netSSID || m.cursor == netPass) {
				break
			}
			max := netMode
			if netModes[m.modeIdx] == "wifi" {
				max = netPass
			}
			if m.cursor < max {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.updateFocus()
			return m, nil

		case "shift+tab", "up", "k":
			if msg.String() == "k" && (m.cursor == netSSID || m.cursor == netPass) {
				break
			}
			if m.cursor > 0 {
				m.cursor--
			} else {
				max := netMode
				if netModes[m.modeIdx] == "wifi" {
					max = netPass
				}
				m.cursor = max
			}
			m.updateFocus()
			return m, nil

		case "left", "h":
			m.modeIdx = (m.modeIdx - 1 + len(netModes)) % len(netModes)
			m.cursor = netMode
			m.updateFocus()
			return m, nil

		case "right", "l":
			m.modeIdx = (m.modeIdx + 1) % len(netModes)
			m.cursor = netMode
			m.updateFocus()
			return m, nil

		case "enter":
			mode := netModes[m.modeIdx]
			if m.cursor == netMode {
				// on the mode row: move down into fields, or proceed if no fields needed
				if mode == "wifi" {
					m.cursor = netSSID
					m.updateFocus()
				} else {
					// ethernet or skip — no fields, just proceed
					m.Save()
					return m, EmitDone()
				}
				return m, nil
			}
			if mode == "wifi" && m.cursor == netSSID {
				m.cursor = netPass
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

		// delegate to focused input
		var cmd tea.Cmd
		if m.cursor == netSSID {
			m.ssid, cmd = m.ssid.Update(msg)
		} else if m.cursor == netPass {
			m.pass, cmd = m.pass.Update(msg)
		}
		return m, cmd
	}
	return m, nil
}

func (m *NetworkModel) updateFocus() {
	m.ssid.Blur()
	m.pass.Blur()
	switch m.cursor {
	case netSSID:
		m.ssid.Focus()
	case netPass:
		m.pass.Focus()
	}
}

func (m NetworkModel) validate() string {
	mode := netModes[m.modeIdx]
	if mode == "wifi" {
		if strings.TrimSpace(m.ssid.Value()) == "" {
			return "SSID cannot be empty"
		}
	}
	return ""
}

func (m NetworkModel) Save() {
	mode := netModes[m.modeIdx]
	m.cfg.NetworkMode = mode
	if mode == "wifi" {
		m.cfg.WifiSSID = strings.TrimSpace(m.ssid.Value())
		m.cfg.WifiPass = m.pass.Value()
	}
}

func (m NetworkModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("02 — NETWORK") + "\n\n")

	// Mode selector
	b.WriteString(style.StyleKey.Render("Mode  "))
	for i, mode := range netModes {
		if i == m.modeIdx {
			b.WriteString(style.ToggleOn(mode))
		} else {
			b.WriteString(style.ToggleOff(mode))
		}
		b.WriteString("  ")
	}
	b.WriteString(style.StyleMuted.Render("  (←→ to change)"))
	b.WriteString("\n\n")

	mode := netModes[m.modeIdx]

	switch mode {
	case "ethernet":
		b.WriteString(style.StyleGood.Render("  ✓ Ethernet detected — no further configuration needed.\n"))
		b.WriteString(style.StyleMuted.Render("  Press enter to continue.\n"))

	case "wifi":
		cursor := func(f netField) string {
			if m.cursor == f {
				return style.StyleSelected.Render("▶ ")
			}
			return "  "
		}
		b.WriteString(cursor(netSSID) + style.StyleKey.Render("SSID      ") + m.ssid.View() + "\n")
		b.WriteString(cursor(netPass) + style.StyleKey.Render("Password  ") + m.pass.View() + "\n")

	case "skip":
		b.WriteString(style.StyleWarn.Render("  ⚠  Network will be skipped.\n"))
		b.WriteString(style.StyleMuted.Render("  pacstrap requires internet. Only skip if you know what you're doing.\n"))
	}

	if m.err != "" {
		b.WriteString("\n" + style.StyleError.Render("  "+m.err) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(style.HelpRow("↑↓/tab", "field", "←→", "mode", "ctrl+a", "advanced", "enter", "next", "esc", "back"))
	return b.String()
}
