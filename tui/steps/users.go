package steps

import (
	"falconia/config"
	"falconia/style"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Step 04: Hostname ────────────────────────────────────────────────────────

// HostnameModel handles hostname input.
type HostnameModel struct {
	cfg   *config.InstallConfig
	input textinput.Model
	err   string
}

var hostnameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]*[a-z0-9])?$`)
var usernameRe = regexp.MustCompile(`^[a-z_][a-z0-9_\-]*$`)

func NewHostname(cfg *config.InstallConfig) HostnameModel {
	ti := textinput.New()
	ti.Placeholder = "berserkarch"
	ti.CharLimit = 63
	ti.Width = 40
	ti.SetValue(cfg.Hostname)
	ti.Focus()
	return HostnameModel{cfg: cfg, input: ti}
}

func (m HostnameModel) Init() tea.Cmd { return m.input.Focus() }

func (m HostnameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "enter":
			v := strings.TrimSpace(m.input.Value())
			if v == "" {
				v = "berserkarch"
			}
			if !hostnameRe.MatchString(v) {
				m.err = "Lowercase alphanumeric and hyphens only; no leading/trailing hyphens"
				return m, nil
			}
			m.cfg.Hostname = v
			return m, EmitDone()
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m HostnameModel) Save() {
	v := strings.TrimSpace(m.input.Value())
	if v == "" {
		v = "berserkarch"
	}
	m.cfg.Hostname = v
}

func (m HostnameModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("04 — HOSTNAME") + "\n\n")
	b.WriteString("  " + style.StyleKey.Render("Hostname  ") + m.input.View() + "\n")
	b.WriteString(style.StyleMuted.Render("  Lowercase letters, numbers, hyphens. No leading/trailing hyphens.\n"))
	if m.err != "" {
		b.WriteString("\n" + style.StyleError.Render("  "+m.err) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(style.HelpRow("ctrl+a", "advanced", "enter", "next", "esc", "back"))
	return b.String()
}

// ── Step 05: Users ───────────────────────────────────────────────────────────

type userField int

const (
	userFieldUsername userField = iota
	userFieldPass
	userFieldPassConfirm
	userFieldShell
	userFieldWheel
	userFieldAdd
)

var shellOptions = []string{"/bin/zsh", "/bin/bash", "/bin/fish"}

// UsersModel handles root password and user creation.
type UsersModel struct {
	cfg *config.InstallConfig

	// root password
	rootPass    textinput.Model
	rootConfirm textinput.Model
	rootDone    bool

	// new user form
	cursor      userField
	username    textinput.Model
	pass        textinput.Model
	passConfirm textinput.Model
	shellIdx    int
	addToWheel  bool
	err         string
}

func NewUsers(cfg *config.InstallConfig) UsersModel {
	rp := textinput.New()
	rp.Placeholder = "root password"
	rp.EchoMode = textinput.EchoPassword
	rp.EchoCharacter = '•'
	rp.Width = 40
	rp.SetValue(cfg.RootPassword)
	rp.Focus()

	rc := textinput.New()
	rc.Placeholder = "confirm"
	rc.EchoMode = textinput.EchoPassword
	rc.EchoCharacter = '•'
	rc.Width = 40
	rc.SetValue(cfg.RootPassword)

	un := textinput.New()
	un.Placeholder = "username"
	un.Width = 35

	up := textinput.New()
	up.Placeholder = "password"
	up.EchoMode = textinput.EchoPassword
	up.EchoCharacter = '•'
	up.Width = 35

	upc := textinput.New()
	upc.Placeholder = "confirm"
	upc.EchoMode = textinput.EchoPassword
	upc.EchoCharacter = '•'
	upc.Width = 35

	shellIdx := 0
	addToWheel := true
	rootDone := false

	if cfg.RootPassword != "" {
		rootDone = true
	}

	if len(cfg.Users) > 0 {
		u := cfg.Users[0]
		un.SetValue(u.Username)
		up.SetValue(u.Password)
		upc.SetValue(u.Password)
		for i, s := range shellOptions {
			if s == u.Shell {
				shellIdx = i
				break
			}
		}
		addToWheel = false
		for _, g := range u.Groups {
			if g == "wheel" {
				addToWheel = true
				break
			}
		}
	}

	m := UsersModel{
		cfg:         cfg,
		rootPass:    rp,
		rootConfirm: rc,
		rootDone:    rootDone,
		username:    un,
		pass:        up,
		passConfirm: upc,
		shellIdx:    shellIdx,
		addToWheel:  addToWheel,
	}

	if rootDone {
		un.Focus()
		m.cursor = userFieldUsername
	}

	return m
}

func (m UsersModel) Init() tea.Cmd {
	if m.rootDone {
		return m.username.Focus()
	}
	return m.rootPass.Focus()
}

func (m UsersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if !m.rootDone {
				return m, EmitBack()
			}
			m.rootDone = false
			return m, nil
		case "tab", "down", "j":
			if !m.rootDone {
				if msg.String() == "j" {
					break
				}
				if m.rootPass.Focused() {
					m.rootPass.Blur()
					m.rootConfirm.Focus()
				} else {
					m.rootConfirm.Blur()
					m.rootPass.Focus()
				}
				return m, nil
			}
			if msg.String() == "j" && (m.cursor == userFieldUsername || m.cursor == userFieldPass || m.cursor == userFieldPassConfirm) {
				break
			}
			if m.cursor < userFieldAdd {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.updateUserFocus()
			return m, nil

		case "shift+tab", "up", "k":
			if !m.rootDone {
				if msg.String() == "k" {
					break
				}
				if m.rootPass.Focused() {
					m.rootPass.Blur()
					m.rootConfirm.Focus()
				} else {
					m.rootConfirm.Blur()
					m.rootPass.Focus()
				}
				return m, nil
			}
			if msg.String() == "k" && (m.cursor == userFieldUsername || m.cursor == userFieldPass || m.cursor == userFieldPassConfirm) {
				break
			}
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = userFieldAdd
			}
			m.updateUserFocus()
			return m, nil

		case "left", "h":
			if m.rootDone && m.cursor == userFieldShell {
				m.shellIdx = (m.shellIdx - 1 + len(shellOptions)) % len(shellOptions)
			}

		case "right", "l":
			if m.rootDone && m.cursor == userFieldShell {
				m.shellIdx = (m.shellIdx + 1) % len(shellOptions)
			}

		case " ":
			if m.rootDone && m.cursor == userFieldWheel {
				m.addToWheel = !m.addToWheel
				return m, nil
			}
		case "enter":
			if !m.rootDone {
				return m.handleRootEnter()
			}
			return m.handleUserEnter()
		}
	}

	var cmd tea.Cmd
	if !m.rootDone {
		if m.rootPass.Focused() {
			m.rootPass, cmd = m.rootPass.Update(msg)
		} else {
			m.rootConfirm, cmd = m.rootConfirm.Update(msg)
		}
	} else {
		switch m.cursor {
		case userFieldUsername:
			m.username, cmd = m.username.Update(msg)
		case userFieldPass:
			m.pass, cmd = m.pass.Update(msg)
		case userFieldPassConfirm:
			m.passConfirm, cmd = m.passConfirm.Update(msg)
		}
	}
	return m, cmd
}

func (m UsersModel) handleRootEnter() (tea.Model, tea.Cmd) {
	if m.rootPass.Focused() {
		m.rootPass.Blur()
		m.rootConfirm.Focus()
		return m, nil
	}
	rp := m.rootPass.Value()
	rc := m.rootConfirm.Value()
	if rp == "" {
		m.err = "Root password cannot be empty"
		return m, nil
	}
	if rp != rc {
		m.err = "Passwords do not match"
		m.rootConfirm.SetValue("")
		m.rootPass.Focus()
		m.rootConfirm.Blur()
		return m, nil
	}
	m.cfg.RootPassword = rp
	m.rootDone = true
	m.err = ""
	m.username.Focus()
	return m, nil
}

func (m UsersModel) handleUserEnter() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case userFieldUsername, userFieldPass, userFieldPassConfirm:
		if m.cursor < userFieldPassConfirm {
			m.cursor++
		} else {
			m.cursor = userFieldShell
		}
		m.updateUserFocus()
		return m, nil

	case userFieldShell:
		m.cursor = userFieldWheel
		m.updateUserFocus()
		return m, nil

	case userFieldWheel:
		m.cursor = userFieldAdd
		m.updateUserFocus()
		return m, nil

	case userFieldAdd:
		// validate and add user
		un := strings.TrimSpace(m.username.Value())
		if un == "" {
			m.err = "Username cannot be empty"
			return m, nil
		}
		if !usernameRe.MatchString(un) {
			m.err = "Invalid username (lowercase letters, digits, hyphens, underscores; must start with a letter or _)"
			return m, nil
		}
		up := m.pass.Value()
		upc := m.passConfirm.Value()
		if up == "" {
			m.err = "User password cannot be empty"
			return m, nil
		}
		if up != upc {
			m.err = "User passwords do not match"
			return m, nil
		}
		groups := []string{"audio", "video", "storage"}
		if m.addToWheel {
			groups = append([]string{"wheel"}, groups...)
		}
		m.cfg.Users = []config.User{{
			Username: un,
			Password: up,
			Shell:    shellOptions[m.shellIdx],
			Groups:   groups,
		}}
		return m, EmitDone()
	}
	return m, nil
}

func (m UsersModel) Save() {
	if m.rootPass.Value() != "" && m.rootPass.Value() == m.rootConfirm.Value() {
		m.cfg.RootPassword = m.rootPass.Value()
	}

	un := strings.TrimSpace(m.username.Value())
	if un != "" {
		groups := []string{"audio", "video", "storage"}
		if m.addToWheel {
			groups = append([]string{"wheel"}, groups...)
		}
		m.cfg.Users = []config.User{{
			Username: un,
			Password: m.pass.Value(),
			Shell:    shellOptions[m.shellIdx],
			Groups:   groups,
		}}
	}
}

func (m *UsersModel) updateUserFocus() {
	m.username.Blur()
	m.pass.Blur()
	m.passConfirm.Blur()
	switch m.cursor {
	case userFieldUsername:
		m.username.Focus()
	case userFieldPass:
		m.pass.Focus()
	case userFieldPassConfirm:
		m.passConfirm.Focus()
	}
}

func (m UsersModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("05 — USERS") + "\n\n")

	if !m.rootDone {
		b.WriteString("  " + style.StyleKey.Render("Root password  ") + m.rootPass.View() + "\n")
		b.WriteString("  " + style.StyleKey.Render("Confirm        ") + m.rootConfirm.View() + "\n")
		if m.err != "" {
			b.WriteString("\n" + style.StyleError.Render("  "+m.err) + "\n")
		}
		b.WriteString("\n")
		b.WriteString(style.HelpRow("↑↓/tab", "field", "ctrl+a", "advanced", "enter", "confirm", "esc", "back"))
		return b.String()
	}

	cursor := func(f userField) string {
		if m.cursor == f {
			return style.StyleSelected.Render("▶ ")
		}
		return "  "
	}

	b.WriteString(style.StyleSubtitle.Render("  Setup your user account") + "\n\n")
	b.WriteString(cursor(userFieldUsername) + style.StyleKey.Render("Username  ") + m.username.View() + "\n")
	b.WriteString(cursor(userFieldPass) + style.StyleKey.Render("Password  ") + m.pass.View() + "\n")
	b.WriteString(cursor(userFieldPassConfirm) + style.StyleKey.Render("Confirm   ") + m.passConfirm.View() + "\n")

	b.WriteString(cursor(userFieldShell) + style.StyleKey.Render("Shell     "))
	for i, s := range shellOptions {
		if i == m.shellIdx {
			b.WriteString(style.ToggleOn(s))
		} else {
			b.WriteString(style.ToggleOff(s))
		}
		b.WriteString("  ")
	}
	b.WriteString("\n")

	b.WriteString(cursor(userFieldWheel) + style.Checkbox(m.addToWheel) + " " + style.StyleKey.Render("sudo (wheel group)") + "\n\n")
	b.WriteString(cursor(userFieldAdd) + style.StyleButtonActive.Render(" Add User ") + "\n")

	if m.err != "" {
		b.WriteString("\n" + style.StyleError.Render("  "+m.err) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(style.HelpRow("↑↓/tab", "field", "←→", "shell", "space", "toggle", "ctrl+a", "advanced", "enter", "confirm", "esc", "back"))
	return b.String()
}
