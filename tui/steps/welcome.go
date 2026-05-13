package steps

import (
	"fmt"

	"falconia/config"
	"falconia/style"

	tea "github.com/charmbracelet/bubbletea"
)

const logo = `
██████╗ ███████╗██████╗ ███████╗███████╗██████╗ ██╗  ██╗ █████╗ ██████╗  ██████╗██╗  ██╗
██╔══██╗██╔════╝██╔══██╗██╔════╝██╔════╝██╔══██╗██║ ██╔╝██╔══██╗██╔══██╗██╔════╝██║  ██║
██████╔╝█████╗  ██████╔╝███████╗█████╗  ██████╔╝█████╔╝ ███████║██████╔╝██║     ███████║
██╔══██╗██╔══╝  ██╔══██╗╚════██║██╔══╝  ██╔══██╗██╔═██╗ ██╔══██║██╔══██╗██║     ██╔══██║
██████╔╝███████╗██║  ██║███████║███████╗██║  ██║██║  ██╗██║  ██║██║  ██║╚██████╗██║  ██║
╚═════╝ ╚══════╝╚═╝  ╚═╝╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝
`

// WelcomeModel is the first step shown to the user.
type WelcomeModel struct {
	cfg      *config.InstallConfig
	firmware string
}

func NewWelcome(cfg *config.InstallConfig) WelcomeModel {
	return WelcomeModel{cfg: cfg, firmware: cfg.Firmware}
}

func (m WelcomeModel) Init() tea.Cmd { return nil }

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ":
			return m, EmitDone()
		}
	}
	return m, nil
}

func (m WelcomeModel) View() string {
	fw := m.firmware
	var fwStr string
	if fw == "uefi" {
		fwStr = style.StyleGood.Render("UEFI")
	} else {
		fwStr = style.StyleWarn.Render("Legacy BIOS")
	}

	out := style.StyleBlue().Render(logo) + "\n"
	out += style.StyleSubtitle.Render("Falconia - Installer for BerserkArch") + "\n\n"
	out += fmt.Sprintf("Detected firmware: %s\n\n", fwStr)
	out += style.StyleMuted.Render("This installer will guide you through a complete BerserkArch") + "\n"
	out += style.StyleMuted.Render("installation. You will configure everything before any") + "\n"
	out += style.StyleMuted.Render("changes are made to your disk.") + "\n\n"
	out += style.StyleWarn.Render("⚠  This will ERASE the selected disk entirely.") + "\n\n"
	out += style.HelpRow("enter", "begin", "ctrl+a", "advanced mode", "q", "quit")
	return out
}
