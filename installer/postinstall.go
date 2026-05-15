package installer

import (
	"os"
	"path/filepath"

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

// InstallPackages installs user-selected extra packages and all service
// dependencies derived from config flags.
func InstallPackages(cfg *config.InstallConfig, log LineHandler) error {
	p := data.New().
		Add(cfg.ExtraPackages...).
		AddIf(cfg.EnableBluetooth, data.ByService["bluetooth"]...).
		AddIf(cfg.EnableCups, data.ByService["cups"]...).
		AddIf(cfg.EnableSSH, data.ByService["ssh"]...)

	for svc, enabled := range cfg.ExtraServices {
		p.AddIf(enabled, data.ByService[svc]...)
	}

	pkgs := p.Build()
	if len(pkgs) == 0 {
		log("No extra packages selected, skipping.")
		return nil
	}

	args := append([]string{"-S", "--noconfirm"}, pkgs...)
	return RunChrootDry(cfg, log, "pacman", args...)
}

// EnableServices enables systemd services based on config flags.
func EnableServices(cfg *config.InstallConfig, log LineHandler) error {
	seen := map[string]struct{}{}
	add := func(units ...string) {
		for _, u := range units {
			seen[u] = struct{}{}
		}
	}

	add(data.AlwaysEnable...)
	if dm, ok := data.DisplayManagerByDE[cfg.DesktopEnv]; ok {
		add(dm)
	}
	if cfg.EnableBluetooth {
		add(data.ByServiceFlag["bluetooth"]...)
	}
	if cfg.EnableCups {
		add(data.ByServiceFlag["cups"]...)
	}
	if cfg.EnableSSH {
		add(data.ByServiceFlag["ssh"]...)
	}
	for svc, on := range cfg.ExtraServices {
		if on {
			add(data.ByServiceFlag[svc]...)
		}
	}

	for svc := range seen {
		if err := RunChrootDry(cfg, log, "systemctl", "enable", svc); err != nil {
			log("Warning: Failed to enable " + svc)
		}
	}

	for _, svc := range data.AlwaysDisable {
		if err := RunChrootDry(cfg, log, "systemctl", "disable", svc); err != nil {
			log("Warning: Failed to disable " + svc)
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
			os.MkdirAll(confDir, 0o755)
			confData := []byte("[zram0]\nzram-size = ram / 2\ncompression-algorithm = zstd\nswap-priority = 100\n")
			if err := os.WriteFile(confPath, confData, 0o644); err != nil {
				log("Warning: Failed to write zram-generator.conf")
			}
		} else {
			log("[DRY RUN] Would write /etc/systemd/zram-generator.conf")
		}
	}

	return nil
}
