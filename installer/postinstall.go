package installer

import (
	"falconia/config"
	"os"
	"path/filepath"
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

// InstallPackages installs the user's selected extra packages and service dependencies.
func InstallPackages(cfg *config.InstallConfig, log LineHandler) error {
	pkgs := append([]string{}, cfg.ExtraPackages...)

	// Append packages required for extra services
	if cfg.ExtraServices["docker"] {
		pkgs = append(pkgs, "docker")
	}
	if cfg.ExtraServices["tailscale"] {
		pkgs = append(pkgs, "tailscale")
	}
	if cfg.ExtraServices["avahi"] {
		pkgs = append(pkgs, "avahi")
	}
	if cfg.ExtraServices["tlp"] {
		pkgs = append(pkgs, "tlp")
	}
	if cfg.ExtraServices["ufw"] {
		pkgs = append(pkgs, "ufw")
	}
	if cfg.ExtraServices["flatpak"] {
		pkgs = append(pkgs, "flatpak")
	}
	if cfg.ExtraServices["snap"] {
		pkgs = append(pkgs, "snapd")
	}
	if cfg.ExtraServices["zram"] {
		pkgs = append(pkgs, "zram-generator")
	}
	if cfg.EnableBluetooth {
		pkgs = append(pkgs, "bluez", "bluez-utils")
	}
	if cfg.EnableCups {
		pkgs = append(pkgs, "cups")
	}
	if cfg.EnableSSH {
		pkgs = append(pkgs, "openssh")
	}

	if len(pkgs) == 0 {
		log("No extra packages selected, skipping.")
		return nil
	}

	args := append([]string{"-S", "--noconfirm"}, pkgs...)
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
	if cfg.ExtraServices["docker"] {
		services = append(services, "docker")
	}
	if cfg.ExtraServices["tailscale"] {
		services = append(services, "tailscaled")
	}
	if cfg.ExtraServices["avahi"] {
		services = append(services, "avahi-daemon")
	}
	if cfg.ExtraServices["tlp"] {
		services = append(services, "tlp")
	}
	if cfg.ExtraServices["ufw"] {
		services = append(services, "ufw")
	}
	if cfg.ExtraServices["snap"] {
		services = append(services, "snapd.socket")
	}
	if cfg.ExtraServices["trim"] {
		services = append(services, "fstrim.timer")
	}

	for _, svc := range services {
		if err := RunChrootDry(cfg, log, "systemctl", "enable", svc); err != nil {
			log("Warning: Failed to enable " + svc)
		}
	}

	// Flatpak configuration
	if cfg.ExtraServices["flatpak"] && !cfg.DryRun {
		log("Configuring Flathub repository...")
		if err := RunChroot(log, "flatpak", "remote-add", "--if-not-exists", "flathub", "https://dl.flathub.org/repo/flathub.flatpakrepo"); err != nil {
			log("Warning: Failed to add Flathub remote")
		}
	} else if cfg.ExtraServices["flatpak"] && cfg.DryRun {
		log("[DRY RUN] Would configure Flathub repository")
	}

	// Zram configuration
	if cfg.ExtraServices["zram"] {
		log("Configuring zram-generator...")
		if !cfg.DryRun {
			confPath := "/mnt/etc/systemd/zram-generator.conf"
			confDir := filepath.Dir(confPath)
			os.MkdirAll(confDir, 0755)
			confData := []byte("[zram0]\nzram-size = ram / 2\ncompression-algorithm = zstd\nswap-priority = 100\n")
			if err := os.WriteFile(confPath, confData, 0644); err != nil {
				log("Warning: Failed to write zram-generator.conf")
			}
		} else {
			log("[DRY RUN] Would write /etc/systemd/zram-generator.conf")
		}
	}

	return nil
}
