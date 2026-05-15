package data

// Enable is the complete list of systemd units to enable on every install.
// Each is attempted unconditionally — if the unit doesn't exist because the
// package wasn't installed, systemctl enable fails silently (warning only).
var Enable = []string{
	// Core system
	"NetworkManager",
	"systemd-timesyncd",
	"power-profiles-daemon",
	"fstrim.timer",
	"auditd",

	// Display managers — only the installed one will succeed
	"gdm",
	"sddm",
	"lightdm",

	// User-installed services — succeed only if the package was selected
	"sshd",
	"cups",
	"bluetooth",
	"docker",
	"tailscaled",
	"avahi-daemon",
	"tlp",
	"ufw",
	"firewalld",
	"snapd.socket",

	// VM guest tools — succeed only inside the matching hypervisor
	"vboxservice",
	"vmtoolsd",
	"vmware-vmblock-fuse",
	"qemu-guest-agent",
	"spice-vdagentd",
}

// Disable is the complete list of systemd units to disable on every install.
var Disable = []string{
	"bluetooth",  // off by default; user enables manually post-install if needed
	"pacman-init", // live-ISO only, must not run on installed system
}
