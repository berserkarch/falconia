package data

// AlwaysEnable are enabled on every installation regardless of config.
// Sourced from BerserkArch Calamares services-systemd.conf.
var AlwaysEnable = []string{
	"NetworkManager",
	"systemd-timesyncd",
	"power-profiles-daemon",
	"fstrim.timer",
	"auditd",
}

// AlwaysDisable are explicitly disabled on every installation.
// bluetooth is off by default; users enable it via the postinstall step.
var AlwaysDisable = []string{
	"bluetooth",
}

// DisplayManagerByDE maps a desktop environment to its display manager service.
var DisplayManagerByDE = map[string]string{
	"kde":      "sddm",
	"gnome":    "sddm",
	"xfce":     "sddm",
	"cinnamon": "lightdm",
	"hyprland": "sddm",
	"i3":       "lightdm",
}

// ByServiceFlag maps ExtraServices keys and the EnableX bool field names to
// the systemd unit(s) they activate.
// Keys that need no service unit (e.g. "microcode") are simply absent.
var ByServiceFlag = map[string][]string{
	"bluetooth": {"bluetooth"},
	"cups":      {"cups"},
	"ssh":       {"sshd"},
	"docker":    {"docker"},
	"tailscale": {"tailscaled"},
	"avahi":     {"avahi-daemon"},
	"tlp":       {"tlp"},
	"ufw":       {"ufw"},
	"firewalld": {"firewalld"},
	"snap":      {"snapd.socket"},
	"trim":      {"fstrim.timer"},
}
