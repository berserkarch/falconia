package installer

import (
	"fmt"

	"falconia/config"
)

// InstallBootloader installs the configured bootloader inside the chroot.
func InstallBootloader(cfg *config.InstallConfig, log LineHandler) error {
	switch cfg.Bootloader {
	case "systemd-boot":
		return installSystemdBoot(cfg, log)
	default:
		return installGrub(cfg, log)
	}
}

func installGrub(cfg *config.InstallConfig, log LineHandler) error {
	// Install GRUB package
	if err := RunChrootDry(cfg, log, "pacman", "-S", "--noconfirm", "grub"); err != nil {
		return err
	}

	if cfg.Firmware == "uefi" {
		if err := RunChrootDry(cfg, log, "pacman", "-S", "--noconfirm", "efibootmgr"); err != nil {
			return err
		}
	}

	// Configure /etc/default/grub BEFORE grub-install.
	// This is important because grub-install on Arch reads this file to
	// decide whether to enable cryptodisk support in the core image.

	// Set GRUB timeout
	if err := RunChrootDry(
		cfg,
		log,
		"sed", "-i",
		fmt.Sprintf(`s/^GRUB_TIMEOUT=.*/GRUB_TIMEOUT=%d/`, cfg.GrubTimeout),
		"/etc/default/grub",
	); err != nil {
		return err
	}

	if cfg.EncryptDisk {
		// Enable cryptodisk support
		if !cfg.DryRun {
			err := RunChroot(log, "bash", "-c", `grep -q "^#\?GRUB_ENABLE_CRYPTODISK=" /etc/default/grub && sed -i 's/^#\?GRUB_ENABLE_CRYPTODISK=.*/GRUB_ENABLE_CRYPTODISK=y/' /etc/default/grub || echo "GRUB_ENABLE_CRYPTODISK=y" >> /etc/default/grub`)
			if err != nil {
				return fmt.Errorf("enable cryptodisk: %w", err)
			}
		} else {
			log(styleGood("[DRY RUN] Would enable GRUB_ENABLE_CRYPTODISK in /etc/default/grub"))
		}

		var uuid string
		if !cfg.DryRun {
			var err error
			uuid, err = runOutput("blkid", "-s", "UUID", "-o", "value", rootPartition(cfg))
			if err != nil {
				return fmt.Errorf("blkid: %w", err)
			}
		} else {
			uuid = "fake-uuid-1234"
		}
		cryptParam := fmt.Sprintf(`cryptdevice=UUID=%s:cryptroot root=/dev/mapper/cryptroot`, uuid)
		if !cfg.DryRun {
			err := RunChroot(log, "sed", "-i", fmt.Sprintf(`s|^\(GRUB_CMDLINE_LINUX_DEFAULT=".*\)"|\1 %s"|`, cryptParam), "/etc/default/grub")
			if err != nil {
				return err
			}
		} else {
			log(styleGood("[DRY RUN] Would add cryptdevice to GRUB_CMDLINE_LINUX_DEFAULT"))
		}
	}

	// Now run grub-install
	if cfg.Firmware == "uefi" {
		if err := RunChrootDry(
			cfg,
			log,
			"grub-install",
			"--target=x86_64-efi",
			"--efi-directory=/boot/efi",
			"--bootloader-id=GRUB",
		); err != nil {
			return fmt.Errorf("grub-install (UEFI): %w", err)
		}
	} else {
		if err := RunChrootDry(
			cfg,
			log,
			"grub-install",
			"--target=i386-pc",
			cfg.Disk,
		); err != nil {
			return fmt.Errorf("grub-install (BIOS): %w", err)
		}
	}

	return RunChrootDry(cfg, log, "grub-mkconfig", "-o", "/boot/grub/grub.cfg")
}

func installSystemdBoot(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.Firmware != "uefi" {
		return fmt.Errorf("systemd-boot requires UEFI firmware")
	}

	if err := RunChrootDry(cfg, log, "bootctl", "--path=/boot/efi", "install"); err != nil {
		return fmt.Errorf("bootctl install: %w", err)
	}

	// For systemd-boot, the kernel and initrd must be on the ESP.
	// We copy them from /boot/ to /boot/efi/
	if !cfg.DryRun {
		vmlinuz := fmt.Sprintf("/boot/vmlinuz-%s", cfg.Kernel)
		initramfs := fmt.Sprintf("/boot/initramfs-%s.img", cfg.Kernel)
		if err := RunChroot(log, "cp", vmlinuz, "/boot/efi/"); err != nil {
			return fmt.Errorf("copy kernel to ESP: %w", err)
		}
		if err := RunChroot(log, "cp", initramfs, "/boot/efi/"); err != nil {
			return fmt.Errorf("copy initramfs to ESP: %w", err)
		}
		// Copy microcode image to ESP if installed; it must also be on the ESP
		// for systemd-boot to load it (unlike GRUB, which handles this automatically).
		if cfg.ExtraServices["microcode"] {
			for _, pkg := range detectMicrocode() {
				src := fmt.Sprintf("/boot/%s.img", pkg)
				if err := RunChroot(log, "cp", src, "/boot/efi/"); err != nil {
					log(fmt.Sprintf("Warning: could not copy %s to ESP: %v", src, err))
				}
			}
		}
	} else {
		log(styleGood("[DRY RUN] Would copy kernel and initramfs to /boot/efi/"))
		if cfg.ExtraServices["microcode"] {
			log(styleGood("[DRY RUN] Would copy microcode image to /boot/efi/"))
		}
	}

	var uuid string
	if !cfg.DryRun {
		var err error
		uuid, err = runOutput("blkid", "-s", "UUID", "-o", "value", rootPartition(cfg))
		if err != nil {
			return fmt.Errorf("blkid: %w", err)
		}
	} else {
		uuid = "fake-uuid-1234"
		log(styleGood("[DRY RUN] Would lookup UUID for: ") + rootPartition(cfg))
	}

	// Build loader entry. Microcode initrd must appear before the main initramfs.
	loaderEntry := fmt.Sprintf("title   BerserkArch\nlinux   /vmlinuz-%s\n", cfg.Kernel)
	if cfg.ExtraServices["microcode"] {
		for _, pkg := range detectMicrocode() {
			loaderEntry += fmt.Sprintf("initrd  /%s.img\n", pkg)
		}
	}
	loaderEntry += fmt.Sprintf("initrd  /initramfs-%s.img\n", cfg.Kernel)

	if cfg.EncryptDisk {
		loaderEntry += fmt.Sprintf("options cryptdevice=UUID=%s:cryptroot root=/dev/mapper/cryptroot rw\n", uuid)
	} else {
		loaderEntry += fmt.Sprintf("options root=UUID=%s rw\n", uuid)
	}

	if !cfg.DryRun {
		if err := writeChroot("/mnt/boot/efi/loader/entries/arch.conf", loaderEntry); err != nil {
			return err
		}
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/boot/efi/loader/entries/arch.conf")
	}

	loaderConf := fmt.Sprintf("default arch\ntimeout %d\nconsole-mode max\neditor no\n", cfg.GrubTimeout)

	if !cfg.DryRun {
		return writeChroot("/mnt/boot/efi/loader/loader.conf", loaderConf)
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/boot/efi/loader/loader.conf")
		return nil
	}
}

// rootPartition returns the root partition path for UUID lookup.
func rootPartition(cfg *config.InstallConfig) string {
	p := partSuffix(cfg.Disk)
	if cfg.SwapSize > 0 {
		return cfg.Disk + p + "3"
	}
	return cfg.Disk + p + "2"
}
