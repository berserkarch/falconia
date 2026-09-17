package installer

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"falconia/config"
	"falconia/data"
)

// RankMirrors runs reflector to rank pacman mirrors by speed.
func RankMirrors(cfg *config.InstallConfig, log LineHandler) error {
	return RunDry(
		cfg, log,
		"reflector",
		"--latest", "20",
		"--sort", "rate",
		"--save", "/etc/pacman.d/mirrorlist",
	)
}

// Pacstrap installs the base system into /mnt.
// --needed makes a re-run (on resume) skip packages that are already installed.
func Pacstrap(cfg *config.InstallConfig, log LineHandler) error {
	pkgs := data.New().
		Add(data.Base...).
		Add(cfg.Kernel, cfg.Kernel+"-headers").
		AddMap(data.ByFilesystem, cfg.Filesystem).
		AddIf(cfg.EncryptDisk, data.Encryption...).
		AddMap(data.ByMicrocode, cfg.Hardware.CPU).
		Build()
	pkgs = withoutExcluded(cfg, log, pkgs)

	args := append([]string{"/mnt", "--needed"}, pkgs...)
	return RunDry(cfg, log, "pacstrap", args...)
}

const cryptoKeyfile = "/mnt/crypto_keyfile.bin"

// AddLuksKeyfile generates a random keyfile and registers it as a second LUKS
// key slot. GRUB prompts for the passphrase once (to read kernel + initramfs
// from the encrypted partition). The initramfs then finds the embedded keyfile
// and unlocks LUKS silently — no second prompt for the user.
//
// Only used with GRUB: for systemd-boot the ESP is unencrypted, so embedding
// the keyfile there would expose it.
//
// Safe to re-run: if a keyfile from a previous attempt already unlocks the
// partition, no new key slot is added.
func AddLuksKeyfile(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would generate /mnt/crypto_keyfile.bin and add as LUKS key slot"))
		return nil
	}

	lukspart := rootPartition(cfg)
	if _, err := os.Stat(cryptoKeyfile); err == nil {
		if Run(nil, "cryptsetup", "open", "--test-passphrase", "--key-file", cryptoKeyfile, lukspart) == nil {
			log("LUKS keyfile already enrolled, skipping")
			return nil
		}
		// Left over from an attempt that failed before luksAddKey; replace it.
		if err := os.Remove(cryptoKeyfile); err != nil {
			return fmt.Errorf("remove stale luks keyfile: %w", err)
		}
	}

	log("Generating LUKS keyfile to eliminate double passphrase prompt...")
	keyfile := make([]byte, 512)
	if _, err := rand.Read(keyfile); err != nil {
		return fmt.Errorf("generate luks keyfile: %w", err)
	}
	// mode 0000: root can still read it (bypasses DAC); normal users cannot.
	if err := os.WriteFile(cryptoKeyfile, keyfile, 0o000); err != nil {
		return fmt.Errorf("write luks keyfile: %w", err)
	}
	// Authorize with the passphrase (no trailing newline, matching luksFormat)
	// to add the keyfile bytes as a new LUKS key slot.
	if err := runWithStdin(
		log, strings.NewReader(cfg.EncryptionPass),
		"cryptsetup", "luksAddKey", "--key-file=-", lukspart, cryptoKeyfile,
	); err != nil {
		return fmt.Errorf("luksAddKey: %w", err)
	}
	return nil
}

// GenerateInitramfs writes the dracut drop-ins, sets the Plymouth theme and
// builds the default and fallback initramfs images.
func GenerateInitramfs(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would execute: ") + "dracut --force /boot/initramfs-" + cfg.Kernel + ".img <kver>")
		return nil
	}

	log("Generating initramfs with dracut...")

	confDir := "/mnt/etc/dracut.conf.d"
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return fmt.Errorf("create dracut conf dir: %w", err)
	}

	// Plymouth must be in dracut modules so the splash screen is embedded in
	// the initramfs. The theme is set below before dracut runs so the right
	// theme assets are baked into the image.
	plymouthConf := "add_dracutmodules+=\" plymouth \"\n"
	if err := os.WriteFile(confDir+"/plymouth.conf", []byte(plymouthConf), 0o644); err != nil {
		log(fmt.Sprintf("Warning: write plymouth dracut config: %v", err))
	}

	// For encrypted installs, ensure dracut includes the crypt module.
	// For GRUB: also embed the keyfile (see AddLuksKeyfile) so the initramfs can
	// open LUKS without a second prompt.
	// For systemd-boot: the initramfs prompts for the passphrase instead (one prompt).
	if cfg.EncryptDisk {
		encConf := "add_dracutmodules+=\" crypt \"\n"
		if cfg.Bootloader != "systemd-boot" {
			encConf += "install_items+=\" /crypto_keyfile.bin \"\n"
		}
		if err := os.WriteFile(confDir+"/encryption.conf", []byte(encConf), 0o644); err != nil {
			return fmt.Errorf("write dracut encryption config: %w", err)
		}
	}

	// Set the Plymouth theme before dracut so the correct theme assets are
	// baked into the initramfs. berserk-plymouth-theme installs the "berserk" theme.
	log("Setting Plymouth theme...")
	if err := RunChroot(log, "plymouth-set-default-theme", "berserk"); err != nil {
		log(fmt.Sprintf("Warning: plymouth-set-default-theme: %v", err))
	}

	// Find the actual kernel version string from /lib/modules.
	// We expect exactly one directory there since we just pacstrapped one kernel.
	kver := detectKver()
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
		log(styleGood("[DRY RUN] Would execute: ") + "genfstab -U /mnt > /mnt/etc/fstab")
		return nil
	}
	// Overwrite rather than append so a re-run (on resume) doesn't duplicate entries.
	log("$ genfstab -U /mnt > /mnt/etc/fstab")
	out, err := runOutput("genfstab", "-U", "/mnt")
	if err != nil {
		return fmt.Errorf("genfstab: %w", err)
	}
	return writeChroot("/mnt/etc/fstab", out+"\n")
}
