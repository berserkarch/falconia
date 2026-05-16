package tui

import (
	"falconia/config"
	"falconia/style"
	"fmt"
	"strings"
)

// SidebarView renders the right-hand summary panel from the current config.
func SidebarView(cfg *config.InstallConfig, step, total int, advanced bool) string {
	var b strings.Builder

	b.WriteString(style.StyleSidebarTitle.Render("SUMMARY") + "\n\n")

	row := func(key, val string) {
		if val == "" {
			val = style.StyleMuted.Render("—")
		}
		b.WriteString(style.StyleKey.Render(fmt.Sprintf("%-16s", key)) + " " + style.StyleValue.Render(val) + "\n")
	}

	// Firmware
	fw := cfg.Firmware
	if fw == "uefi" {
		fw = style.StyleGood.Render("UEFI")
	} else if fw == "bios" {
		fw = style.StyleWarn.Render("BIOS")
	}
	row("Firmware", fw)

	// Disk
	diskLabel := cfg.Disk
	if cfg.DiskModel != "" {
		diskLabel = cfg.DiskModel
	}
	row("Disk", diskLabel)
	if cfg.Disk != "" {
		row("Scheme", cfg.PartitionScheme)
		row("FS", cfg.Filesystem)
		switch cfg.SwapMode {
		case "partition":
			row("Swap", fmt.Sprintf("partition (%d MiB)", cfg.SwapSize))
		case "file":
			row("Swap", fmt.Sprintf("file (%d MiB)", cfg.SwapSize))
		case "suspend":
			row("Swap", "file (auto, hibernate)")
		default:
			row("Swap", "none")
		}
	}

	b.WriteString("\n")

	// Network
	switch cfg.NetworkMode {
	case "wifi":
		row("Network", "WiFi: "+cfg.WifiSSID)
	case "ethernet":
		row("Network", style.StyleGood.Render("Ethernet"))
	case "skip":
		row("Network", style.StyleWarn.Render("Skipped"))
	default:
		row("Network", "")
	}

	b.WriteString("\n")

	// Locale
	row("Timezone", cfg.Timezone)
	row("Locale", cfg.Locale)
	row("Keymap", cfg.Keymap)

	b.WriteString("\n")

	// Identity
	row("Hostname", cfg.Hostname)

	// Users
	if len(cfg.Users) > 0 {
		names := make([]string, len(cfg.Users))
		for i, u := range cfg.Users {
			names[i] = u.Username
		}
		row("Users", strings.Join(names, ", "))
	} else {
		row("Users", "")
	}

	b.WriteString("\n")

	// Software
	row("Kernel", cfg.Kernel)
	row("Desktop", cfg.DesktopEnv)
	row("Bootloader", cfg.Bootloader)

	b.WriteString("\n")

	if cfg.RankMirrors {
		row("Mirrors", "rank on install")
	}
	if len(cfg.ExtraPackages) > 0 {
		row("Packages", fmt.Sprintf("%d selected", len(cfg.ExtraPackages)))
	}

	b.WriteString("\n")

	// Step counter
	b.WriteString(style.StyleMuted.Render(fmt.Sprintf("Step %d / %d", step, total)) + "\n")

	// Mode badge
	if advanced {
		b.WriteString(style.StyleModeAdvanced.Render(" advanced "))
	} else {
		b.WriteString(style.StyleModeSimple.Render(" simple "))
	}

	return style.StyleSidebar.Render(b.String())
}
