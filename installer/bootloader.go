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

	// Set GRUB timeout in /etc/default/grub
	if err := RunChrootDry(
		cfg,
		log,
		"sed", "-i",
		fmt.Sprintf(`s/^GRUB_TIMEOUT=.*/GRUB_TIMEOUT=%d/`, cfg.GrubTimeout),
		"/etc/default/grub",
	); err != nil {
		return err
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

	loaderEntry := fmt.Sprintf(
		"title   BerserkArch\nlinux   /vmlinuz-%s\ninitrd  /initramfs-%s.img\noptions root=UUID=%s rw\n",
		cfg.Kernel, cfg.Kernel, uuid,
	)

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
	if cfg.SwapSize > 0 || cfg.Firmware == "uefi" {
		return cfg.Disk + p + "3"
	}
	return cfg.Disk + p + "2"
}
