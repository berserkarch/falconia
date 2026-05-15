package steps

import (
	"fmt"
	"strings"

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
	hw := m.cfg.Hardware

	fwStr := style.StyleGood.Render("UEFI")
	if m.firmware != "uefi" {
		fwStr = style.StyleWarn.Render("Legacy BIOS")
	}

	row := func(label, value string) string {
		return fmt.Sprintf("  %s  %s\n",
			style.StyleKey.Render(fmt.Sprintf("%-12s", label)),
			style.StyleValue.Render(value))
	}

	out := style.StyleBlue().Render(logo) + "\n"
	out += style.StyleSubtitle.Render("Falconia — Installer for BerserkArch") + "\n\n"

	out += row("Firmware", fwStr)

	// CPU + microcode
	cpuLabel := strings.ToUpper(hw.CPU)
	if hw.CPU == "other" || hw.CPU == "" {
		cpuLabel = "Unknown"
	}
	ucodeNote := ""
	switch hw.CPU {
	case "intel":
		ucodeNote = style.StyleMuted.Render("  → intel-ucode")
	case "amd":
		ucodeNote = style.StyleMuted.Render("  → amd-ucode")
	}
	out += row("CPU", cpuLabel+ucodeNote)

	// RAM
	if hw.RAMBytes > 0 {
		out += row("RAM", formatRAM(hw.RAMBytes))
	}

	// GPUs
	if len(hw.GPUs) == 0 {
		out += row("GPU", style.StyleMuted.Render("none detected"))
	}
	for i, gpu := range hw.GPUs {
		label := "GPU"
		if i > 0 {
			label = ""
		}
		model := gpu.Model
		if model == "" {
			model = gpu.Vendor
		}
		var driverNote string
		switch gpu.Vendor {
		case "nvidia":
			if gpu.UseOpen {
				driverNote = style.StyleMuted.Render("  → nvidia-open")
			} else {
				driverNote = style.StyleMuted.Render("  → nvidia")
			}
		case "amd":
			driverNote = style.StyleMuted.Render("  → amdgpu (built-in)")
		case "intel":
			driverNote = style.StyleMuted.Render("  → i915 (built-in)")
		}
		out += row(label, model+driverNote)
	}

	// WiFi
	if hw.WiFi == "broadcom" {
		out += row("WiFi", style.StyleWarn.Render("Broadcom  → broadcom-wl"))
	}

	// NVMe
	if hw.HasNVMe {
		out += row("NVMe", style.StyleGood.Render("detected  → nvme_load=YES"))
	}

	// VM
	if hw.VM != "none" && hw.VM != "" {
		out += row("Environment", style.StyleWarn.Render(capitalize(hw.VM)+" VM detected"))
	}

	out += "\n"
	out += style.StyleMuted.Render("Configure everything before any changes are made to your disk.") + "\n"
	out += style.StyleWarn.Render("⚠  This will ERASE the selected disk entirely.") + "\n\n"
	out += style.HelpRow("enter", "begin", "ctrl+a", "advanced mode", "q", "quit")
	return out
}

func formatRAM(bytes int64) string {
	gib := float64(bytes) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.1f GiB", gib)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
