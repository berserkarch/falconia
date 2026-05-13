package installer

import (
	"falconia/config"
	"fmt"
	"os"
	"strings"
)

// RankMirrors runs reflector to rank pacman mirrors by speed.
func RankMirrors(cfg *config.InstallConfig, log LineHandler) error {
	return RunDry(cfg, log,
		"reflector",
		"--latest", "20",
		"--sort", "rate",
		"--save", "/etc/pacman.d/mirrorlist",
	)
}

// Pacstrap installs the base system into /mnt.
func Pacstrap(cfg *config.InstallConfig, log LineHandler) error {
	pkgs := []string{
		"base",
		"base-devel",
		cfg.Kernel,
		cfg.Kernel + "-headers",
		"linux-firmware",
		"networkmanager",
		"sudo",
		"vim",
	}

	// Add filesystem tools
	switch cfg.Filesystem {
	case "btrfs":
		pkgs = append(pkgs, "btrfs-progs")
	case "xfs":
		pkgs = append(pkgs, "xfsprogs")
	default:
		pkgs = append(pkgs, "e2fsprogs")
	}

	// Add microcode
	pkgs = append(pkgs, detectMicrocode()...)

	args := append([]string{"/mnt"}, pkgs...)
	return RunDry(cfg, log, "pacstrap", args...)
}

// detectMicrocode returns the appropriate microcode package based on CPU vendor.
func detectMicrocode() []string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil
	}
	content := string(data)
	if strings.Contains(content, "GenuineIntel") {
		return []string{"intel-ucode"}
	}
	if strings.Contains(content, "AuthenticAMD") {
		return []string{"amd-ucode"}
	}
	return nil
}

// GenFstab generates /mnt/etc/fstab.
func GenFstab(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would execute: ") + "genfstab -U /mnt >> /mnt/etc/fstab")
		return nil
	}
	log("$ genfstab -U /mnt >> /mnt/etc/fstab")
	out, err := runOutput("genfstab", "-U", "/mnt")
	if err != nil {
		return fmt.Errorf("genfstab: %w", err)
	}
	return appendFile("/mnt/etc/fstab", out)
}
