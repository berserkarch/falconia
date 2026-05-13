package installer

import (
	"falconia/config"
)

// InstallDesktop installs the chosen DE and its display manager.
func InstallDesktop(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DesktopEnv == "none" {
		log("No desktop environment selected, skipping.")
		return nil
	}

	pkgs := config.DEPackages(cfg.DesktopEnv)
	if len(pkgs) == 0 {
		return nil
	}

	args := append([]string{"-S", "--noconfirm"}, pkgs...)
	return RunChrootDry(cfg, log, "pacman", args...)
}

// InstallPackages installs the user's selected extra packages.
func InstallPackages(cfg *config.InstallConfig, log LineHandler) error {
	if len(cfg.ExtraPackages) == 0 {
		log("No extra packages selected, skipping.")
		return nil
	}

	args := append([]string{"-S", "--noconfirm"}, cfg.ExtraPackages...)
	return RunChrootDry(cfg, log, "pacman", args...)
}

// EnableServices enables systemd services based on config flags.
func EnableServices(cfg *config.InstallConfig, log LineHandler) error {
	services := []string{"NetworkManager"} // always

	if cfg.EnableBluetooth {
		services = append(services, "bluetooth")
	}
	if cfg.EnableCups {
		services = append(services, "cups")
	}
	if cfg.EnableSSH {
		services = append(services, "sshd")
	}
	if cfg.DesktopEnv != "none" && cfg.DisplayManager != "" {
		services = append(services, cfg.DisplayManager)
	}

	for _, svc := range services {
		if err := RunChrootDry(cfg, log, "systemctl", "enable", svc); err != nil {
			return err
		}
	}
	return nil
}
