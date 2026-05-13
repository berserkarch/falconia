package installer

import (
	"falconia/config"
	"fmt"
	"strings"
)

// SetTimezone sets the system timezone inside the chroot.
func SetTimezone(cfg *config.InstallConfig, log LineHandler) error {
	link := fmt.Sprintf("/usr/share/zoneinfo/%s", cfg.Timezone)
	if err := RunChrootDry(cfg, log, "ln", "-sf", link, "/etc/localtime"); err != nil {
		return err
	}
	return RunChrootDry(cfg, log, "hwclock", "--systohc")
}

// SetLocale writes /etc/locale.gen and generates locales.
func SetLocale(cfg *config.InstallConfig, log LineHandler) error {
	localeGen := cfg.Locale + " UTF-8\n"
	if !cfg.DryRun {
		if err := writeChroot("/mnt/etc/locale.gen", localeGen); err != nil {
			return fmt.Errorf("write locale.gen: %w", err)
		}
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/etc/locale.gen")
	}

	if err := RunChrootDry(cfg, log, "locale-gen"); err != nil {
		return err
	}

	localeConf := fmt.Sprintf("LANG=%s\n", cfg.Locale)
	if !cfg.DryRun {
		if err := writeChroot("/mnt/etc/locale.conf", localeConf); err != nil {
			return fmt.Errorf("write locale.conf: %w", err)
		}
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/etc/locale.conf")
	}

	vconsole := fmt.Sprintf("KEYMAP=%s\n", cfg.Keymap)
	if !cfg.DryRun {
		return writeChroot("/mnt/etc/vconsole.conf", vconsole)
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/etc/vconsole.conf")
		return nil
	}
}

// ConfigureNetwork sets up the Wi-Fi connection for NetworkManager.
func ConfigureNetwork(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.NetworkMode != "wifi" || cfg.WifiSSID == "" {
		return nil // Nothing to configure
	}

	log("Configuring Wi-Fi for SSID: " + cfg.WifiSSID)

	if cfg.DryRun {
		log("[DRY RUN] Would write NetworkManager connection profile for " + cfg.WifiSSID)
		return nil
	}

	// Create NetworkManager connection profile directory
	if err := RunSh(log, "mkdir -p /mnt/etc/NetworkManager/system-connections"); err != nil {
		return err
	}

	// Write the connection profile
	profile := fmt.Sprintf(`[connection]
id=%s
uuid=11111111-1111-1111-1111-111111111111
type=wifi

[wifi]
mode=infrastructure
ssid=%s

[wifi-security]
key-mgmt=wpa-psk
psk=%s

[ipv4]
method=auto

[ipv6]
addr-gen-mode=default
method=auto
`, cfg.WifiSSID, cfg.WifiSSID, cfg.WifiPass)

	path := fmt.Sprintf("/mnt/etc/NetworkManager/system-connections/%s.nmconnection", cfg.WifiSSID)
	if err := writeChroot(path, profile); err != nil {
		return err
	}

	// Restrict permissions (NetworkManager requires 600)
	return RunSh(log, fmt.Sprintf("chmod 600 %q", path))
}

// SetHostname writes /etc/hostname and /etc/hosts.
func SetHostname(cfg *config.InstallConfig, log LineHandler) error {
	if !cfg.DryRun {
		log(fmt.Sprintf("$ echo '%s' > /etc/hostname", cfg.Hostname))
		if err := writeChroot("/mnt/etc/hostname", cfg.Hostname+"\n"); err != nil {
			return err
		}
	} else {
		log(styleGood("[DRY RUN] Would execute: ") + "echo '" + cfg.Hostname + "' > /mnt/etc/hostname")
	}

	hosts := fmt.Sprintf(
		"127.0.0.1\tlocalhost\n::1\t\tlocalhost\n127.0.1.1\t%s.localdomain\t%s\n",
		cfg.Hostname, cfg.Hostname,
	)
	if !cfg.DryRun {
		return writeChroot("/mnt/etc/hosts", hosts)
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/etc/hosts")
		return nil
	}
}

// SetRootPassword sets the root password inside the chroot.
func SetRootPassword(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would execute: ") + "arch-chroot /mnt passwd (setting root password)")
		return nil
	}
	log("$ setting root password...")
	input := strings.NewReader(cfg.RootPassword + "\n" + cfg.RootPassword + "\n")
	return runWithStdin(log, input, "arch-chroot", "/mnt", "passwd")
}

// CreateUsers creates all non-root users defined in config.
func CreateUsers(cfg *config.InstallConfig, log LineHandler) error {
	for _, u := range cfg.Users {
		groups := strings.Join(u.Groups, ",")
		if err := RunChrootDry(cfg, log,
			"useradd",
			"-m",
			"-G", groups,
			"-s", u.Shell,
			u.Username,
		); err != nil {
			return fmt.Errorf("useradd %s: %w", u.Username, err)
		}

		if cfg.DryRun {
			log(styleGood("[DRY RUN] Would execute: ") + "arch-chroot /mnt passwd " + u.Username)
		} else {
			input := strings.NewReader(u.Password + "\n" + u.Password + "\n")
			if err := runWithStdin(log, input, "arch-chroot", "/mnt", "passwd", u.Username); err != nil {
				return fmt.Errorf("passwd %s: %w", u.Username, err)
			}
		}
	}

	// Uncomment %wheel in sudoers
	return RunChrootDry(cfg, log,
		"sed", "-i",
		`s/^# %wheel ALL=(ALL:ALL) ALL/%wheel ALL=(ALL:ALL) ALL/`,
		"/etc/sudoers",
	)
}
