package installer

import (
	"crypto/rand"
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
		// For GRUB: also embed the keyfile so the initramfs can open LUKS without a
		// second prompt (GRUB already prompted once to read the kernel from /boot).
		// For systemd-boot: the ESP is unencrypted, so embedding the keyfile there
		// would expose it; the initramfs prompts for the passphrase instead (one prompt).
		if cfg.EncryptDisk {
			confDir := "/mnt/etc/dracut.conf.d"
			os.MkdirAll(confDir, 0755)
			encConf := "add_dracutmodules+=\" crypt \"\n"
			if cfg.Bootloader != "systemd-boot" {
				encConf += "install_items+=\" /crypto_keyfile.bin \"\n"
			}
			if err := os.WriteFile(confDir+"/encryption.conf", []byte(encConf), 0644); err != nil {
				return fmt.Errorf("write dracut encryption config: %w", err)
			}

			if cfg.Bootloader != "systemd-boot" {
				// Generate a random keyfile and register it as a second LUKS key slot.
				// GRUB prompts for the passphrase once (to read kernel + initramfs from the
				// encrypted partition). The initramfs then finds the embedded keyfile and
				// unlocks LUKS silently — no second prompt for the user.
				log("Generating LUKS keyfile to eliminate double passphrase prompt...")
				keyfile := make([]byte, 512)
				if _, err := rand.Read(keyfile); err != nil {
					return fmt.Errorf("generate luks keyfile: %w", err)
				}
				// mode 0000: root can still read it (bypasses DAC); normal users cannot.
				if err := os.WriteFile("/mnt/crypto_keyfile.bin", keyfile, 0000); err != nil {
					return fmt.Errorf("write luks keyfile: %w", err)
				}
				// Authorize with the passphrase (no trailing newline, matching luksFormat)
				// to add the keyfile bytes as a new LUKS key slot.
				lukspart := rootPartition(cfg)
				if err := runWithStdin(log, strings.NewReader(cfg.EncryptionPass),
					"cryptsetup", "luksAddKey", "--key-file=-", lukspart, "/mnt/crypto_keyfile.bin",
				); err != nil {
					return fmt.Errorf("luksAddKey: %w", err)
				}
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
		if cfg.EncryptDisk && cfg.Bootloader != "systemd-boot" {
			log(styleGood("[DRY RUN] Would generate /mnt/crypto_keyfile.bin and add as LUKS key slot"))
		}
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

	// For GRUB: keyfile is embedded in the initramfs; systemd-cryptsetup reads it
	// to unlock LUKS without a second prompt.
	// For systemd-boot: ESP is unencrypted so keyfile isn't embedded; "none" tells
	// systemd-cryptsetup-generator to prompt for the passphrase instead.
	passField := "/crypto_keyfile.bin"
	if cfg.Bootloader == "systemd-boot" {
		passField = "none"
	}
	content := "# <name>\t<device>\t\t\t\t<password>\t\t\t<options>\n"
	content += fmt.Sprintf("cryptroot\tUUID=%s\t\t%s\tluks\n", uuid, passField)
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
