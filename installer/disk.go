package installer

import (
	"falconia/config"
	"fmt"
	"os"
	"strings"
)

// PartitionDisk wipes and partitions cfg.Disk according to firmware and scheme.
func PartitionDisk(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.PartitionScheme == "manual" {
		log(styleGood("[INFO] Using manual partitioning, skipping automatic partition creation"))
		return nil
	}
	disk := cfg.Disk

	// Wipe existing signatures
	if err := RunDry(cfg, log, "wipefs", "-a", disk); err != nil {
		return fmt.Errorf("wipefs: %w", err)
	}

	if cfg.Firmware == "uefi" {
		return partitionUEFI(cfg, log)
	}
	return partitionBIOS(cfg, log)
}

// partitionUEFI creates:
//
//	sdX1 – 512MiB EFI System (FAT32)
//	sdX2 – SwapSize MiB Linux swap  (if SwapSize > 0)
//	sdX3 – remainder Linux filesystem
func partitionUEFI(cfg *config.InstallConfig, log LineHandler) error {
	disk := cfg.Disk

	cmds := [][]string{
		{"parted", "-s", disk, "mklabel", "gpt"},
		{"parted", "-s", disk, "mkpart", "ESP", "fat32", "1MiB", "513MiB"},
		{"parted", "-s", disk, "set", "1", "esp", "on"},
	}

	next := "513MiB"
	partNum := 2

	if cfg.SwapSize > 0 {
		swapEnd := fmt.Sprintf("%dMiB", 513+cfg.SwapSize)
		cmds = append(cmds,
			[]string{"parted", "-s", disk, "mkpart", "swap", "linux-swap",
				next, swapEnd},
		)
		next = swapEnd
		partNum++
	}

	cmds = append(cmds,
		[]string{"parted", "-s", disk, "mkpart", "root", cfg.Filesystem, next, "100%"},
	)
	_ = partNum

	for _, c := range cmds {
		if err := RunDry(cfg, log, c[0], c[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// partitionBIOS creates:
//
//	sdX1 – 1MiB BIOS boot (no fs)
//	sdX2 – SwapSize MiB Linux swap  (if SwapSize > 0)
//	sdX3 – remainder Linux filesystem
func partitionBIOS(cfg *config.InstallConfig, log LineHandler) error {
	disk := cfg.Disk

	cmds := [][]string{
		{"parted", "-s", disk, "mklabel", "msdos"},
		{"parted", "-s", disk, "mkpart", "primary", "1MiB", "2MiB"},
		{"parted", "-s", disk, "set", "1", "bios_grub", "on"},
	}

	next := "2MiB"

	if cfg.SwapSize > 0 {
		swapEnd := fmt.Sprintf("%dMiB", 2+cfg.SwapSize)
		cmds = append(cmds,
			[]string{"parted", "-s", disk, "mkpart", "primary", "linux-swap", next, swapEnd},
		)
		next = swapEnd
	}

	cmds = append(cmds,
		[]string{"parted", "-s", disk, "mkpart", "primary", next, "100%"},
	)

	for _, c := range cmds {
		if err := RunDry(cfg, log, c[0], c[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// partSuffix returns the partition suffix for a given disk.
// NVMe devices use "p" (e.g. nvme0n1p1), others use nothing (e.g. sda1).
func partSuffix(disk string) string {
	// nvme/mmcblk devices need a "p" before partition number
	for _, prefix := range []string{"/dev/nvme", "/dev/mmcblk"} {
		if len(disk) > len(prefix) && disk[:len(prefix)] == prefix {
			return "p"
		}
	}
	return ""
}

// FormatDisks mkfs's each partition according to config.
func FormatDisks(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.PartitionScheme == "manual" {
		for mnt, part := range cfg.MountPoints {
			var mkfsCmd []string
			if mnt == "/boot/efi" {
				mkfsCmd = []string{"mkfs.fat", "-F32", part}
			} else if mnt == "swap" {
				mkfsCmd = []string{"mkswap", part}
			} else {
				switch cfg.Filesystem {
				case "btrfs":
					mkfsCmd = []string{"mkfs.btrfs", "-f", part}
				case "xfs":
					mkfsCmd = []string{"mkfs.xfs", "-f", part}
				default: // ext4
					mkfsCmd = []string{"mkfs.ext4", "-F", part}
				}
			}
			if err := RunDry(cfg, log, mkfsCmd[0], mkfsCmd[1:]...); err != nil {
				return fmt.Errorf("format %s (%s): %w", mnt, part, err)
			}
		}
		return nil
	}

	disk := cfg.Disk
	p := partSuffix(disk)

	if cfg.Firmware == "uefi" {
		// EFI partition
		if err := RunDry(cfg, log, "mkfs.fat", "-F32", disk+p+"1"); err != nil {
			return fmt.Errorf("format EFI: %w", err)
		}
	}

	rootPart := disk + p + "3"
	swapPart := disk + p + "2"

	if cfg.SwapSize > 0 {
		if err := RunDry(cfg, log, "mkswap", swapPart); err != nil {
			return fmt.Errorf("mkswap: %w", err)
		}
	} else {
		// no swap: root is always partition 2 regardless of firmware
		// (BIOS: bios-boot=1, root=2; UEFI: esp=1, root=2)
		rootPart = disk + p + "2"
	}

	var mkfsCmd []string

	if cfg.EncryptDisk {
		log("Encrypting root partition with LUKS...")
		if !cfg.DryRun {
			// Pass passphrase via stdin without a trailing newline.
			// cryptsetup -d - reads stdin until EOF and stores every byte as the key.
			// If a newline is included here it becomes part of the key, but the boot-time
			// password prompt strips Enter before sending to cryptsetup open → mismatch.
			if err := runWithStdin(log, strings.NewReader(cfg.EncryptionPass),
				"cryptsetup", "-q", "luksFormat", "--type", "luks1", rootPart, "-d", "-"); err != nil {
				return fmt.Errorf("luksFormat: %w", err)
			}
			if err := runWithStdin(log, strings.NewReader(cfg.EncryptionPass),
				"cryptsetup", "open", rootPart, "cryptroot", "-d", "-"); err != nil {
				return fmt.Errorf("luksOpen: %w", err)
			}
		} else {
			log(styleGood("[DRY RUN] Would execute: ") + "cryptsetup luksFormat " + rootPart)
			log(styleGood("[DRY RUN] Would execute: ") + "cryptsetup open " + rootPart + " cryptroot")
		}
		rootPart = "/dev/mapper/cryptroot"
	}

	switch cfg.Filesystem {
	case "btrfs":
		mkfsCmd = []string{"mkfs.btrfs", "-f", rootPart}
	case "xfs":
		mkfsCmd = []string{"mkfs.xfs", "-f", rootPart}
	default: // ext4
		mkfsCmd = []string{"mkfs.ext4", "-F", rootPart}
	}

	if err := RunDry(cfg, log, mkfsCmd[0], mkfsCmd[1:]...); err != nil {
		return fmt.Errorf("format root: %w", err)
	}
	return nil
}

// MountDisks mounts the partitions under /mnt.
func MountDisks(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.PartitionScheme == "manual" {
		// Always mount root first
		if rootPart, ok := cfg.MountPoints["/"]; ok {
			if err := RunDry(cfg, log, "mount", rootPart, "/mnt"); err != nil {
				return fmt.Errorf("mount root: %w", err)
			}
		} else {
			return fmt.Errorf("manual partitioning: root (/) partition not found")
		}

		// Mount others
		for mnt, part := range cfg.MountPoints {
			if mnt == "/" || mnt == "swap" {
				continue
			}
			fullMnt := "/mnt" + mnt
			if !cfg.DryRun {
				if err := os.MkdirAll(fullMnt, 0755); err != nil {
					return fmt.Errorf("mkdir %s: %w", mnt, err)
				}
			} else {
				log(styleGood("[DRY RUN] Would execute: ") + "mkdir -p " + fullMnt)
			}
			if err := RunDry(cfg, log, "mount", part, fullMnt); err != nil {
				return fmt.Errorf("mount %s: %w", mnt, err)
			}
		}

		// Swap
		if swapPart, ok := cfg.MountPoints["swap"]; ok {
			if err := RunDry(cfg, log, "swapon", swapPart); err != nil {
				return fmt.Errorf("swapon: %w", err)
			}
		}
		return nil
	}

	disk := cfg.Disk
	p := partSuffix(disk)

	rootPart := disk + p + "3"
	swapPart := disk + p + "2"

	if cfg.SwapSize == 0 {
		rootPart = disk + p + "2"
	}

	if cfg.EncryptDisk {
		rootPart = "/dev/mapper/cryptroot"
	}

	// Mount root
	if err := RunDry(cfg, log, "mount", rootPart, "/mnt"); err != nil {
		return fmt.Errorf("mount root: %w", err)
	}

	// Mount EFI
	if cfg.Firmware == "uefi" {
		efiMount := "/mnt/boot/efi"
		if !cfg.DryRun {
			if err := os.MkdirAll(efiMount, 0755); err != nil {
				return fmt.Errorf("mkdir efi: %w", err)
			}
		} else {
			log(styleGood("[DRY RUN] Would execute: ") + "mkdir -p " + efiMount)
		}
		if err := RunDry(cfg, log, "mount", disk+p+"1", efiMount); err != nil {
			return fmt.Errorf("mount EFI: %w", err)
		}
	}

	// Enable swap
	if cfg.SwapSize > 0 {
		if err := RunDry(cfg, log, "swapon", swapPart); err != nil {
			return fmt.Errorf("swapon: %w", err)
		}
	}

	return nil
}

// Cleanup unmounts everything under /mnt and disables swap.
func Cleanup(cfg *config.InstallConfig, log LineHandler) error {
	_ = RunDry(cfg, log, "swapoff", "-a")
	err := RunDry(cfg, log, "umount", "-R", "/mnt")
	if cfg.EncryptDisk && !cfg.DryRun {
		_ = RunDry(cfg, log, "cryptsetup", "close", "cryptroot")
	} else if cfg.EncryptDisk && cfg.DryRun {
		log(styleGood("[DRY RUN] Would execute: ") + "cryptsetup close cryptroot")
	}
	return err
}
