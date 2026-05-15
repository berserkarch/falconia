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
		"dracut",
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

	if cfg.EncryptDisk {
		pkgs = append(pkgs, "cryptsetup")
	}

	// Add microcode
	if cfg.ExtraServices["microcode"] {
		pkgs = append(pkgs, detectMicrocode()...)
	}

	args := append([]string{"/mnt"}, pkgs...)
	if err := RunDry(cfg, log, "pacstrap", args...); err != nil {
		return err
	}

	if !cfg.DryRun {
		// Generate initramfs with dracut
		log("Generating initramfs with dracut...")

		// For encrypted installs, ensure dracut includes the crypt module.
		// Running inside arch-chroot on a non-encrypted live ISO means dracut
		// cannot auto-detect the target's dm-crypt root; we must be explicit.
		if cfg.EncryptDisk {
			confDir := "/mnt/etc/dracut.conf.d"
			os.MkdirAll(confDir, 0755)
			encConf := `add_dracutmodules+=" crypt "` + "\n"
			if err := os.WriteFile(confDir+"/encryption.conf", []byte(encConf), 0644); err != nil {
				return fmt.Errorf("write dracut encryption config: %w", err)
			}
		}

		// Find the actual kernel version string from /lib/modules.
		// We expect exactly one directory there since we just pacstrapped one kernel.
		files, err := os.ReadDir("/mnt/lib/modules")
		kver := ""
		if err == nil && len(files) > 0 {
			kver = files[0].Name()
		}

		if kver == "" {
			// Fallback to kernel name if we can't detect, though dracut might fail
			kver = cfg.Kernel
		}

		outputPath := fmt.Sprintf("/boot/initramfs-%s.img", cfg.Kernel)
		if err := RunChroot(log, "dracut", "--force", outputPath, kver); err != nil {
			return fmt.Errorf("dracut: %w", err)
		}

		// Generate a generic fallback initramfs. Pacstrap's mkinitcpio hook creates
		// initramfs-<kernel>-fallback.img without the crypt module, so we overwrite it
		// with a dracut generic image (--no-hostonly) that includes all drivers including
		// dm-crypt. grub-mkconfig generates a fallback boot entry for this file; without
		// this step that entry would fail to unlock root on encrypted installs.
		log("Generating fallback initramfs with dracut...")
		fallbackPath := fmt.Sprintf("/boot/initramfs-%s-fallback.img", cfg.Kernel)
		if err := RunChroot(log, "dracut", "--no-hostonly", "--force", fallbackPath, kver); err != nil {
			log(fmt.Sprintf("Warning: dracut fallback: %v", err))
		}
	} else {
		log(styleGood("[DRY RUN] Would execute: ") + "dracut --force /boot/initramfs-" + cfg.Kernel + ".img <kver>")
	}

	return nil
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

// GenCrypttab writes /mnt/etc/crypttab for LUKS-encrypted installs.
// The running system's systemd-cryptsetup-generator reads this file; it is
// also required by some post-install tools (e.g. cryptsetup-initramfs helpers)
// to know which devices are encrypted and under what mapper names.
func GenCrypttab(cfg *config.InstallConfig, log LineHandler) error {
	if !cfg.EncryptDisk {
		return nil
	}
	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would write: ") + "/mnt/etc/crypttab")
		return nil
	}

	uuid, err := runOutput("blkid", "-s", "UUID", "-o", "value", rootPartition(cfg))
	if err != nil {
		return fmt.Errorf("blkid for crypttab: %w", err)
	}

	content := "# <name>\t<device>\t\t\t\t<password>\t<options>\n"
	content += fmt.Sprintf("cryptroot\tUUID=%s\t\tnone\t\tluks\n", uuid)
	log("$ writing /mnt/etc/crypttab")
	return writeChroot("/mnt/etc/crypttab", content)
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
