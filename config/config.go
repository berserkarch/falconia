// Package config just contains configs
package config

// HardwareProfile is populated once at startup by installer.DetectHardware()
// and never changes after that.  All installer functions read from it.
type HardwareProfile struct {
	CPU      string // "intel" | "amd" | "other"
	GPUs     []GPUInfo
	WiFi     string // "broadcom" | "other"
	VM       string // "virtualbox" | "vmware" | "kvm" | "qemu" | "none"
	RAMBytes int64  // total physical RAM in bytes
	HasNVMe  bool
}

// GPUInfo describes one graphics adapter found on the system.
type GPUInfo struct {
	Vendor  string // "nvidia" | "amd" | "intel" | "other"
	Model   string // human-readable model name from lspci
	UseOpen bool   // true = install nvidia-open; false = nvidia (proprietary)
}

// InstallConfig is the single source of truth passed through every step.
// All Phase 1 steps read from and write to this struct.
// Phase 2 installer functions take a *InstallConfig and execute accordingly.
type InstallConfig struct {
	// --- Hardware (auto-detected at startup, never user-set) ---
	Hardware HardwareProfile

	// --- System ---
	Firmware string // "uefi" | "bios"  (auto-detected on startup)

	// --- Disk ---
	Disk            string // e.g. "/dev/sda"
	DiskModel       string // human label e.g. "Samsung SSD 870 (500 GB)"
	PartitionScheme string // "guided" | "manual"
	Filesystem      string // "ext4" | "btrfs" | "xfs"
	SwapMode        string // "none" | "partition" | "file" | "suspend"
	SwapSize        int    // MiB; used for partition/file modes; ignored for suspend (auto-sized to RAM)
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
	DesktopEnv string // "none" | "gnome" | "kde" | "xfce" | "hyprland" | "i3"

	// --- Extra Packages ---
	ExtraPackages []string

	// --- Bootloader ---
	Bootloader  string // "grub" | "systemd-boot"
	GrubTimeout int    // seconds (GRUB only)

	// --- Post-install ---
	RankMirrors    bool
	WindowsEFIPath string // non-empty if a Windows installation was detected

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
		SwapMode:        "partition",
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
	}
}

// BootloaderOptions returns valid bootloader choices for the detected firmware.
func (c *InstallConfig) BootloaderOptions() []string {
	if c.Firmware == "uefi" {
		return []string{"grub", "systemd-boot"}
	}
	return []string{"grub"}
}

