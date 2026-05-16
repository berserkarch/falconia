package installer

import (
	"fmt"
	"os"
	"strings"

	"falconia/config"
)

// SetupSwap creates a swap file for "file" and "suspend" modes.
// It must run after MountDisks so /mnt is available.
func SetupSwap(cfg *config.InstallConfig, log LineHandler) error {
	swapPath := "/mnt/swapfile"

	sizeMiB := cfg.SwapSize
	if cfg.SwapMode == "suspend" {
		// Auto-size: RAM + 1 GiB headroom so the hibernation image fits.
		ramMiB := int(cfg.Hardware.RAMBytes >> 20)
		if ramMiB < 1024 {
			// RAM detection failed (or impossibly small system); use a generous
			// fallback so hibernation has room to write the resume image.
			log("Warning: RAM size unknown or implausibly small; using 8 GiB fallback for hibernation swap")
			ramMiB = 8192
		}
		sizeMiB = ramMiB + 1024
	}
	if sizeMiB <= 0 {
		sizeMiB = 4096
	}

	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would create: ") + fmt.Sprintf("/swapfile (%d MiB, %s)", sizeMiB, cfg.Filesystem))
		log(styleGood("[DRY RUN] Would execute: ") + "swapon /mnt/swapfile")
		if cfg.SwapMode == "suspend" {
			log(styleGood("[DRY RUN] Would write: ") + "/etc/dracut.conf.d/resume.conf")
		}
		return nil
	}

	if cfg.Filesystem == "btrfs" {
		// btrfs mkswapfile handles COW disable, permissions, and mkswap in one shot.
		if err := Run(log, "btrfs", "filesystem", "mkswapfile",
			"--size", fmt.Sprintf("%dM", sizeMiB), swapPath); err != nil {
			return fmt.Errorf("btrfs mkswapfile: %w", err)
		}
	} else {
		if err := Run(log, "fallocate", "-l", fmt.Sprintf("%dMiB", sizeMiB), swapPath); err != nil {
			return fmt.Errorf("fallocate swapfile: %w", err)
		}
		if err := os.Chmod(swapPath, 0o600); err != nil {
			return fmt.Errorf("chmod swapfile: %w", err)
		}
		if err := Run(log, "mkswap", swapPath); err != nil {
			return fmt.Errorf("mkswap: %w", err)
		}
	}

	if err := Run(log, "swapon", swapPath); err != nil {
		return fmt.Errorf("swapon: %w", err)
	}

	if cfg.SwapMode == "suspend" {
		return setupResume(cfg, log, swapPath)
	}
	return nil
}

// setupResume writes a dracut drop-in that enables hibernate resume from the
// swap file. Must run after the swap file exists and swapon is active.
func setupResume(cfg *config.InstallConfig, log LineHandler, swapPath string) error {
	// resumeDev is what the kernel mounts as the container of the swap file.
	// LUKS: point at the unlocked mapper device (UUID of the raw partition would
	// pre-date decryption). Plain: UUID of the root partition itself.
	var resumeDev string
	if cfg.EncryptDisk {
		resumeDev = "/dev/mapper/cryptroot"
	} else {
		uuid, err := runOutput("blkid", "-s", "UUID", "-o", "value", rootPartition(cfg))
		if err != nil {
			return fmt.Errorf("blkid UUID: %w", err)
		}
		resumeDev = "UUID=" + strings.TrimSpace(uuid)
	}

	var offset string
	if cfg.Filesystem == "btrfs" {
		out, err := runOutput("btrfs", "inspect-internal", "map-swapfile", "-r", swapPath)
		if err != nil {
			return fmt.Errorf("btrfs map-swapfile: %w", err)
		}
		offset = strings.TrimSpace(out)
	} else {
		out, err := runOutput("filefrag", "-v", swapPath)
		if err != nil {
			return fmt.Errorf("filefrag: %w", err)
		}
		offset, err = parseFilefragOffset(out)
		if err != nil {
			return fmt.Errorf("parse swap file offset: %w", err)
		}
	}

	conf := fmt.Sprintf(
		"add_dracutmodules+=\" resume \"\nkernel_cmdline+=\" resume=%s resume_offset=%s \"\n",
		resumeDev, offset,
	)

	confDir := "/mnt/etc/dracut.conf.d"
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return fmt.Errorf("mkdir dracut.conf.d: %w", err)
	}
	if err := writeChroot(confDir+"/resume.conf", conf); err != nil {
		return fmt.Errorf("write resume.conf: %w", err)
	}
	log("$ wrote /etc/dracut.conf.d/resume.conf (resume=" + resumeDev + " resume_offset=" + offset + ")")
	return nil
}

// parseFilefragOffset extracts the physical offset of the first extent from
// filefrag -v output. The offset is in filesystem blocks and is used directly
// as the kernel's resume_offset parameter.
func parseFilefragOffset(out string) (string, error) {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		// First data line starts with "0:" — fourth field is "NNNN.."
		if len(fields) >= 4 && fields[0] == "0:" {
			return strings.TrimRight(fields[3], "."), nil
		}
	}
	return "", fmt.Errorf("no extents found in filefrag output")
}
