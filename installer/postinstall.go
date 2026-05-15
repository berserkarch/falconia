package installer

import (
	"falconia/config"
	"falconia/data"
)

// InstallDesktop installs the chosen DE and its display manager.
func InstallDesktop(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DesktopEnv == "none" {
		log("No desktop environment selected, skipping.")
		return nil
	}

	pkgs := data.New().AddMap(data.ByDE, cfg.DesktopEnv).Build()
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

// EnableServices enables and disables systemd units from data/services.go.
// Each enable is attempted unconditionally — if the unit doesn't exist
// because the package wasn't installed, it fails as a warning and moves on.
func EnableServices(cfg *config.InstallConfig, log LineHandler) error {
	for _, svc := range data.Enable {
		if err := RunChrootDry(cfg, log, "systemctl", "enable", svc); err != nil {
			log("Warning: could not enable " + svc)
		}
	}

	for _, svc := range data.Disable {
		if err := RunChrootDry(cfg, log, "systemctl", "disable", svc); err != nil {
			log("Warning: could not disable " + svc)
		}
	}

	return RunChrootDry(cfg, log, "systemctl", "set-default", "graphical.target")
}
