package installer

import (
	"crypto/rand"
	"falconia/config"
	"fmt"
	"os"
	"strings"
)

// SetTimezone sets the system timezone inside the chroot.
func SetTimezone(cfg *config.InstallConfig, log LineHandler) error {
	link := fmt.Sprintf("/usr/share/zoneinfo/%s", cfg.Timezone)
	if err := RunChrootDry(cfg, log, "ln", "-sf", link, "/etc/localtime"); err != nil {
		return err
	}
	// Write /etc/timezone as a plain-text fallback; the symlink at /etc/localtime
	// is authoritative on Arch, but some tools (tzdata, Python's zoneinfo module)
	// read this file when the symlink target is ambiguous.
	if !cfg.DryRun {
		if err := writeChroot("/mnt/etc/timezone", cfg.Timezone+"\n"); err != nil {
			return fmt.Errorf("write /etc/timezone: %w", err)
		}
	} else {
		log(styleGood("[DRY RUN] Would write file: ") + "/mnt/etc/timezone")
	}
	// Sync the hardware clock; fall back to the ISA bus method for machines whose
	// kernel RTC driver is broken or absent (some older or embedded hardware).
	if err := RunChrootDry(cfg, log, "hwclock", "--systohc"); err != nil {
		log("Warning: hwclock --systohc failed, retrying via ISA bus...")
		return RunChrootDry(cfg, log, "hwclock", "--systohc", "--directisa")
	}
	return nil
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

	// Write the connection profile.
	// Only include [wifi-security] when a password is provided — omitting it
	// entirely is how NetworkManager handles open (unencrypted) access points.
	security := ""
	if cfg.WifiPass != "" {
		security = fmt.Sprintf("[wifi-security]\nkey-mgmt=wpa-psk\npsk=%s\n\n", cfg.WifiPass)
	}
	profile := fmt.Sprintf(`[connection]
id=%s
uuid=%s
type=wifi

[wifi]
mode=infrastructure
ssid=%s

%s[ipv4]
method=auto

[ipv6]
addr-gen-mode=default
method=auto
`, cfg.WifiSSID, randomUUID(), cfg.WifiSSID, security)

	path := fmt.Sprintf("/mnt/etc/NetworkManager/system-connections/%s.nmconnection", cfg.WifiSSID)
	if err := writeChroot(path, profile); err != nil {
		return err
	}

	// Restrict permissions (NetworkManager requires 600)
	return os.Chmod(path, 0600)
}

func randomUUID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
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

// existingGroups returns the set of group names present in /mnt/etc/group.
func existingGroups() map[string]struct{} {
	data, err := os.ReadFile("/mnt/etc/group")
	if err != nil {
		return nil
	}
	groups := make(map[string]struct{})
	for _, line := range strings.Split(string(data), "\n") {
		if name, _, found := strings.Cut(line, ":"); found {
			groups[name] = struct{}{}
		}
	}
	return groups
}

// CreateUsers creates all non-root users defined in config.
func CreateUsers(cfg *config.InstallConfig, log LineHandler) error {
	known := existingGroups()

	for _, u := range cfg.Users {
		var filtered []string
		for _, g := range u.Groups {
			if known == nil {
				filtered = append(filtered, g)
				continue
			}
			if _, ok := known[g]; ok {
				filtered = append(filtered, g)
			} else {
				log(styleGood("[INFO] ") + "skipping group '" + g + "' — not present in chroot")
			}
		}
		groups := strings.Join(filtered, ",")
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
