package installer

import (
	"fmt"
	"os"

	"falconia/config"
	"falconia/data"
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

	// Build base kernel params: splash + hardware-driven flags.
	baseParams := "quiet splash"
	if cfg.Filesystem == "btrfs" && cfg.PartitionScheme != "manual" {
		// Guided btrfs uses the @ subvolume layout. Without this, the kernel
		// mounts subvolid=5 (top-level) and can't find /sbin/init.
		// Manual mode skips this because we don't create subvolumes there.
		baseParams += " rootflags=subvol=/@"
	}
	if cfg.Hardware.HasNVMe {
		baseParams += " nvme_load=YES"
	}
	for _, gpu := range cfg.Hardware.GPUs {
		if gpu.Vendor == "nvidia" {
			baseParams += " nvidia-drm.modeset=1"
			break
		}
	}
	if !cfg.DryRun {
		err := RunChroot(log, "sed", "-i",
			fmt.Sprintf(`s|^\(GRUB_CMDLINE_LINUX_DEFAULT=".*\)"|\1 %s"|`, baseParams),
			"/etc/default/grub")
		if err != nil {
			log(fmt.Sprintf("Warning: add base params to GRUB cmdline: %v", err))
		}
	} else {
		log(styleGood("[DRY RUN] Would add to GRUB_CMDLINE_LINUX_DEFAULT: ") + baseParams)
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
		// dracut reads rd.luks.uuid/rd.luks.name, NOT cryptdevice= (mkinitcpio syntax).
		// rd.luks.name maps the UUID to the /dev/mapper/cryptroot name used by root=.
		cryptParam := fmt.Sprintf(`rd.luks.uuid=%s rd.luks.name=%s=cryptroot rd.luks.key=/crypto_keyfile.bin root=/dev/mapper/cryptroot`, uuid, uuid)
		if !cfg.DryRun {
			err := RunChroot(log, "sed", "-i", fmt.Sprintf(`s|^\(GRUB_CMDLINE_LINUX_DEFAULT=".*\)"|\1 %s"|`, cryptParam), "/etc/default/grub")
			if err != nil {
				return err
			}
		} else {
			log(styleGood("[DRY RUN] Would add rd.luks params to GRUB_CMDLINE_LINUX_DEFAULT"))
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
		// Install to the standard removable path (EFI/BOOT/BOOTX64.EFI) as well.
		// Many UEFI firmware implementations ignore NVRAM boot entries entirely and
		// only scan the removable fallback path, so this makes the system bootable
		// on that hardware without any manual NVRAM manipulation.
		if err := RunChrootDry(
			cfg,
			log,
			"grub-install",
			"--target=x86_64-efi",
			"--efi-directory=/boot/efi",
			"--bootloader-id=GRUB",
			"--removable",
		); err != nil {
			log(fmt.Sprintf("Warning: grub-install removable fallback: %v", err))
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

	// If Windows was detected, tell grub-mkconfig to include it via os-prober.
	if cfg.WindowsEFIPath != "" {
		if !cfg.DryRun {
			err := RunChroot(log, "bash", "-c",
				`grep -q "^#\?GRUB_DISABLE_OS_PROBER=" /etc/default/grub `+
					`&& sed -i 's/^#\?GRUB_DISABLE_OS_PROBER=.*/GRUB_DISABLE_OS_PROBER=false/' /etc/default/grub `+
					`|| echo "GRUB_DISABLE_OS_PROBER=false" >> /etc/default/grub`)
			if err != nil {
				log("Warning: enable os-prober for dual-boot: " + err.Error())
			}
		} else {
			log(styleGood("[DRY RUN] Would enable GRUB_DISABLE_OS_PROBER=false for dual-boot"))
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
		// Copy microcode image to ESP — required for systemd-boot to load it
		// (unlike GRUB which handles microcode automatically from /boot).
		ucodePkgs := data.ByMicrocode[cfg.Hardware.CPU]
		for _, pkg := range ucodePkgs {
			src := fmt.Sprintf("/boot/%s.img", pkg)
			if err := RunChroot(log, "cp", src, "/boot/efi/"); err != nil {
				log(fmt.Sprintf("Warning: could not copy %s to ESP: %v", src, err))
			}
		}
	} else {
		log(styleGood("[DRY RUN] Would copy kernel and initramfs to /boot/efi/"))
		if ucode := data.ByMicrocode[cfg.Hardware.CPU]; len(ucode) > 0 {
			log(styleGood("[DRY RUN] Would copy microcode to /boot/efi/: ") + ucode[0])
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
	for _, pkg := range data.ByMicrocode[cfg.Hardware.CPU] {
		loaderEntry += fmt.Sprintf("initrd  /%s.img\n", pkg)
	}
	loaderEntry += fmt.Sprintf("initrd  /initramfs-%s.img\n", cfg.Kernel)

	var kernelOpts string
	extraParams := "quiet splash"
	if cfg.Filesystem == "btrfs" && cfg.PartitionScheme != "manual" {
		extraParams += " rootflags=subvol=/@"
	}
	if cfg.Hardware.HasNVMe {
		extraParams += " nvme_load=YES"
	}
	for _, gpu := range cfg.Hardware.GPUs {
		if gpu.Vendor == "nvidia" {
			extraParams += " nvidia-drm.modeset=1"
			break
		}
	}
	if cfg.EncryptDisk {
		// No rd.luks.key here: the keyfile is not embedded in the initramfs for
		// systemd-boot (ESP is unencrypted). The crypt module will prompt once.
		kernelOpts = fmt.Sprintf("rd.luks.uuid=%s rd.luks.name=%s=cryptroot root=/dev/mapper/cryptroot rw %s", uuid, uuid, extraParams)
	} else {
		kernelOpts = fmt.Sprintf("root=UUID=%s rw %s", uuid, extraParams)
	}
	loaderEntry += "options " + kernelOpts + "\n"

	if !cfg.DryRun {
		if err := writeChroot("/mnt/boot/efi/loader/entries/arch.conf", loaderEntry); err != nil {
			return err
		}
		// Write /etc/kernel/cmdline so future kernel-install invocations (triggered by
		// kernel package upgrades) generate new boot entries with the correct parameters.
		// Without this file, updated kernels would boot without LUKS/root params.
		os.MkdirAll("/mnt/etc/kernel", 0755)
		if err := writeChroot("/mnt/etc/kernel/cmdline", kernelOpts+"\n"); err != nil {
			log("Warning: could not write /etc/kernel/cmdline: " + err.Error())
		}
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/boot/efi/loader/entries/arch.conf")
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/etc/kernel/cmdline")
	}

	// Windows dual-boot entry for systemd-boot.
	if cfg.WindowsEFIPath != "" {
		windowsEntry := "title   Windows\nefi     /EFI/Microsoft/Boot/bootmgfw.efi\n"
		if !cfg.DryRun {
			if err := writeChroot("/mnt/boot/efi/loader/entries/windows.conf", windowsEntry); err != nil {
				log("Warning: write Windows boot entry: " + err.Error())
			} else {
				log("$ wrote /boot/efi/loader/entries/windows.conf")
			}
		} else {
			log(styleGood("[DRY RUN] Would write: ") + "/mnt/boot/efi/loader/entries/windows.conf")
		}
	}

	loaderConf := fmt.Sprintf("default arch\ntimeout %d\nconsole-mode max\neditor no\n", cfg.GrubTimeout)

	if !cfg.DryRun {
		return writeChroot("/mnt/boot/efi/loader/loader.conf", loaderConf)
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/boot/efi/loader/loader.conf")
		return nil
	}
}

// rootPartition returns the block device path for the root partition.
// Manual mode: the user-picked partition for "/".
// Guided + swap partition: p3 (boot/bios=1, swap=2, root=3).
// Guided otherwise:        p2 (boot/bios=1, root=2).
func rootPartition(cfg *config.InstallConfig) string {
	if cfg.PartitionScheme == "manual" {
		return cfg.MountPoints["/"]
	}
	p := partSuffix(cfg.Disk)
	if cfg.SwapMode == "partition" {
		return cfg.Disk + p + "3"
	}
	return cfg.Disk + p + "2"
}
