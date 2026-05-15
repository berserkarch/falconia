// Package config just contains configs
package config

// InstallConfig is the single source of truth passed through every step.
// All Phase 1 steps read from and write to this struct.
// Phase 2 installer functions take a *InstallConfig and execute accordingly.
type InstallConfig struct {
	// --- System ---
	Firmware string // "uefi" | "bios"  (auto-detected on startup)

	// --- Disk ---
	Disk            string // e.g. "/dev/sda"
	DiskModel       string // human label e.g. "Samsung SSD 870 (500 GB)"
	PartitionScheme string // "guided" | "manual"
	Filesystem      string // "ext4" | "btrfs" | "xfs"
	SwapSize        int    // MiB; 0 = no swap partition
	EncryptDisk     bool
	EncryptionPass  string // never written to disk or logged

	// --- Network ---
	NetworkMode string // "wifi" | "ethernet" | "skip"
	WifiSSID    string
	WifiPass    string // never written to disk or logged

	// --- Locale ---
	Timezone string // e.g. "Asia/Kolkata"
	Locale   string // e.g. "en_US.UTF-8"
	Keymap   string // e.g. "us"

	// --- Identity ---
	Hostname string

	// --- Users ---
	RootPassword string // never logged
	Users        []User

	// --- Kernel ---
	Kernel string // "linux" | "linux-lts" | "linux-zen" | "linux-hardened"

	// --- Desktop ---
	DesktopEnv     string // "none" | "gnome" | "kde" | "xfce" | "hyprland" | "i3"
	DisplayManager string // auto-set from DE choice

	// --- Extra Packages ---
	ExtraPackages []string

	// --- Bootloader ---
	Bootloader  string // "grub" | "systemd-boot"
	GrubTimeout int    // seconds (GRUB only)

	// --- Post-install ---
	EnableBluetooth bool
	EnableCups      bool
	EnableSSH       bool
	RankMirrors     bool
	ExtraServices   map[string]bool // dynamic services

	// --- Manual Partitioning ---
	MountPoints map[string]string // mount point -> partition (e.g. "/" -> "/dev/sda3")

	// --- Debugging ---
	DryRun bool
}

// User represents a non-root user to be created.
type User struct {
	Username string
	Password string   // never logged
	Shell    string   // "/bin/bash" | "/bin/zsh" | "/bin/fish"
	Groups   []string // e.g. ["wheel", "audio", "video"]
}

// Defaults returns a config pre-filled with sensible defaults.
func Defaults() *InstallConfig {
	return &InstallConfig{
		PartitionScheme: "guided",
		Filesystem:      "ext4",
		SwapSize:        4096,
		Locale:          "en_US.UTF-8",
		Keymap:          "us",
		Kernel:          "linux",
		DesktopEnv:      "none",
		Bootloader:      "grub",
		GrubTimeout:     5,
		RankMirrors:     true,
		Hostname:        "berserkarch",
		MountPoints:     make(map[string]string),
		ExtraServices:   make(map[string]bool),
	}
}

// BootloaderOptions returns valid bootloader choices for the detected firmware.
func (c *InstallConfig) BootloaderOptions() []string {
	if c.Firmware == "uefi" {
		return []string{"grub", "systemd-boot"}
	}
	return []string{"grub"}
}

// DEDisplayManager maps a desktop environment to its default display manager.
func DEDisplayManager(de string) string {
	switch de {
	case "gnome":
		return "gdm"
	case "kde", "hyprland", "i3":
		return "sddm"
	case "xfce":
		return "sddm"
	default:
		return ""
	}
}

// DEPackages returns the pacman package group(s) for a given DE.
func DEPackages(de string) []string {
	switch de {
	case "gnome":
		return []string{"gnome", "gnome-extra", "gdm"}
	case "kde":
		return []string{"plasma", "plasma-desktop", "plasma-meta", "sddm", "berserk-user-config", "berserk-config-kde"}
	case "xfce":
		return []string{"xfce4", "xfce4-goodies", "lightdm", "lightdm-gtk-greeter"}
	case "hyprland":
		return []string{
			"hyprland", "xdg-desktop-portal-hyprland", "waybar", "sddm",
			"kitty", "wofi", "mako", "polkit-kde-agent",
		}
	case "i3":
		return []string{
			"i3-wm", "i3status", "i3lock", "dmenu", "sddm",
			"xorg", "xterm", "picom", "nitrogen",
		}
	default:
		return nil
	}
}
