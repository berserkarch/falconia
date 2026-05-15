package installer

import (
	"os"

	"falconia/config"
	"falconia/data"
)

// InstallDesktop installs the chosen DE and its display manager.
func InstallDesktop(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DesktopEnv == "none" {
		log("No desktop environment selected, skipping.")
		return nil
	}

	pkgs := data.New().
		Add(data.CommonDE...).
		AddMap(data.ByDE, cfg.DesktopEnv).
		Build()
	if len(pkgs) == 0 {
		return nil
	}

	args := append([]string{"-S", "--noconfirm"}, pkgs...)
	return RunChrootDry(cfg, log, "pacman", args...)
}

// InstallPackages installs user-selected extra packages.
func InstallPackages(cfg *config.InstallConfig, log LineHandler) error {
	pkgs := data.New().Add(cfg.ExtraPackages...).Build()
	if len(pkgs) == 0 {
		log("No extra packages selected, skipping.")
		return nil
	}

	args := append([]string{"-S", "--noconfirm"}, pkgs...)
	return RunChrootDry(cfg, log, "pacman", args...)
}

// PostInstallCleanup performs final housekeeping on the installed system.
func PostInstallCleanup(cfg *config.InstallConfig, log LineHandler) error {
	// Replace the live system's machine ID with a fresh one for the new install.
	if err := RunChrootDry(cfg, log, "systemd-machine-id-setup"); err != nil {
		log("Warning: systemd-machine-id-setup: " + err.Error())
	}

	// Delete .pacnew files left behind when pacstrap writes config files that
	// already exist (usually because we copied them from the live environment).
	if !cfg.DryRun {
		if err := Run(log, "find", "/mnt", "-name", "*.pacnew", "-delete"); err != nil {
			log("Warning: .pacnew cleanup: " + err.Error())
		}
	} else {
		log(styleGood("[DRY RUN] Would remove: ") + "*.pacnew files under /mnt")
	}

	// Enable persistent journal storage so logs survive reboots.
	confDir := "/mnt/etc/systemd/journald.conf.d"
	if !cfg.DryRun {
		if err := os.MkdirAll(confDir, 0o755); err != nil {
			log("Warning: create journald.conf.d: " + err.Error())
			return nil
		}
		if err := writeChroot(confDir+"/00-persistence.conf", "[Journal]\nStorage=persistent\n"); err != nil {
			log("Warning: write journald persistence config: " + err.Error())
		} else {
			log("$ wrote /etc/systemd/journald.conf.d/00-persistence.conf")
		}
	} else {
		log(styleGood("[DRY RUN] Would write: ") + "/etc/systemd/journald.conf.d/00-persistence.conf")
	}

	return nil
}

// EnableServices enables and disables systemd units from data/services.go.
// Each enable is attempted unconditionally — if the unit doesn't exist
// because the package wasn't installed, it fails as a warning and moves on.
func EnableServices(cfg *config.InstallConfig, log LineHandler) error {
	for _, svc := range data.Enable {
		if err := RunChrootDry(cfg, log, "systemctl", "enable", svc); err != nil {
			log("Warning: could not enable " + svc + ": " + err.Error())
		}
	}

	for _, svc := range data.Disable {
		if err := RunChrootDry(cfg, log, "systemctl", "disable", svc); err != nil {
			log("Warning: could not disable " + svc + ": " + err.Error())
		}
	}

	return RunChrootDry(cfg, log, "systemctl", "set-default", "graphical.target")
}
