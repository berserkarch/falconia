package data

// PackageList is a fluent builder for assembling pacman package lists
// conditionally. Duplicates are silently dropped at Build time.
//
// Usage:
//
//	pkgs := data.New().
//	    Add(data.Base...).
//	    AddIf(cfg.EncryptDisk, data.Encryption...).
//	    AddMap(data.ByFilesystem, cfg.Filesystem).
//	    Build()
type PackageList struct {
	pkgs []string
}

func New() *PackageList { return &PackageList{} }

// Add appends packages unconditionally.
func (p *PackageList) Add(pkgs ...string) *PackageList {
	p.pkgs = append(p.pkgs, pkgs...)
	return p
}

// AddIf appends packages only when cond is true.
func (p *PackageList) AddIf(cond bool, pkgs ...string) *PackageList {
	if cond {
		p.pkgs = append(p.pkgs, pkgs...)
	}
	return p
}

// AddMap looks up key in m and appends the resulting slice.
// No-op if the key is absent.
func (p *PackageList) AddMap(m map[string][]string, key string) *PackageList {
	if pkgs, ok := m[key]; ok {
		p.pkgs = append(p.pkgs, pkgs...)
	}
	return p
}

// Build returns a deduplicated slice preserving insertion order.
func (p *PackageList) Build() []string {
	seen := make(map[string]struct{}, len(p.pkgs))
	out := make([]string, 0, len(p.pkgs))
	for _, pkg := range p.pkgs {
		if _, dup := seen[pkg]; !dup {
			seen[pkg] = struct{}{}
			out = append(out, pkg)
		}
	}
	return out
}

// ── Package lists ────────────────────────────────────────────────────────────

// Base is installed on every system via pacstrap.
var Base = []string{
	"base",
	"base-devel",
	"linux-firmware",
	"networkmanager",
	"dracut",
	"sudo",
	"vim",
	"device-mapper",
	"diffutils",
	"dosfstools",
	"efibootmgr",
	"exfatprogs",
	"f2fs-tools",
	"iptables-nft",
	"inetutils",
	"jfsutils",
	"less",
	"logrotate",
	"lsb-release",
	"lvm2",
	"man-db",
	"man-pages",
	"mdadm",
	"nano",
	"netctl",
	"ntfs-3g",
	"perl",
	"python",
	"s-nail",
	"sysfsutils",
	"systemd-sysvcompat",
	"texinfo",
	"usbutils",
	"which",
	"xterm",
	"eza",
	// BerserkArch keyrings + mirrorlists
	"archlinux-keyring",
	"berserk-keyring",
	"blackarch-keyring",
	"chaotic-keyring",
	"berserk-mirrorlist",
	"blackarch-mirrorlist",
	"chaotic-mirrorlist",
	// BerserkArch meta packages
	"plymouth",
	"python-pipx",
	"berserk-hooks",
	"berserk-rofi",
	"berserk-default-themes",
	"berserk-grub-theme",
	"berserk-omz",
	"berserk-gtk-theme",
	"berserk-user-config",
	"berserk-wallpapers",
	"berserk-plymouth-theme",
	"berserk-sunity-cursor",
	"berserk-tpm-tmux",
	"berserk-lazyvim",
	"berserk-apparmor",
	"berserk-firefox-defaults",
	"berserk-neofetch",
	"berserk-btweak",
	"berserk",
}

// Encryption is added when LUKS is enabled.
var Encryption = []string{"cryptsetup"}

// ByFilesystem maps the chosen filesystem to its userspace tools.
var ByFilesystem = map[string][]string{
	"ext4":  {"e2fsprogs"},
	"btrfs": {"btrfs-progs"},
	"xfs":   {"xfsprogs"},
}

// ByMicrocode maps a detected CPU vendor to its microcode package.
// Populated once hardware detection (Phase C) is wired in; used via AddMap.
var ByMicrocode = map[string][]string{
	"intel": {"intel-ucode"},
	"amd":   {"amd-ucode"},
}

// ByDE maps a desktop environment name to its pacman packages.
// Package lists sourced from the BerserkArch Calamares packagechooser.conf.
var ByDE = map[string][]string{
	"kde": {
		"ark", "bluedevil", "breeze-gtk", "dolphin", "dolphin-plugins",
		"ffmpegthumbs", "fwupd", "gwenview", "haruna", "kate", "kcalc",
		"kde-cli-tools", "kde-gtk-config", "kdeconnect",
		"kdegraphics-thumbnailers", "kdenetwork-filesharing",
		"kdeplasma-addons", "kgamma", "kimageformats", "kinfocenter",
		"kio-admin", "kio-extras", "kio-fuse", "konsole", "kscreen",
		"kwallet-pam", "kwayland-integration", "libappindicator", "okular",
		"plasma-browser-integration", "plasma-desktop", "plasma-disks",
		"plasma-firewall", "plasma-keyboard", "plasma-nm", "plasma-pa",
		"plasma-systemmonitor", "plasma-workspace", "plasma-x11-session",
		"powerdevil", "print-manager", "qt6-imageformats", "sddm",
		"sddm-kcm", "spectacle", "xdg-desktop-portal-kde", "xsettingsd",
		"berserk-sddm-theme", "berserk-user-config", "berserk-config-kde",
	},
	"gnome": {
		"adwaita-icon-theme", "loupe", "evince", "file-roller", "gdm",
		"gnome-calculator", "gnome-clocks", "gnome-console",
		"gnome-control-center", "gnome-disk-utility", "gnome-keyring",
		"gnome-nettool", "gnome-power-manager", "gnome-shell",
		"gnome-system-monitor", "gnome-terminal", "gnome-text-editor",
		"gnome-themes-extra", "gnome-tweaks", "gnome-usage", "gnome-weather",
		"gvfs", "gvfs-afc", "gvfs-gphoto2", "gvfs-mtp", "gvfs-nfs",
		"gvfs-smb", "nautilus", "sushi", "showtime",
		"xdg-desktop-portal-gnome", "xdg-desktop-portal",
		"xdg-user-dirs-gtk", "gnome-shell-extensions",
		"gnome-shell-extension-dash-to-dock", "sddm",
		"berserk-config-gnome",
	},
	"xfce": {
		"blueman", "file-roller", "galculator", "gvfs", "gvfs-afc",
		"gvfs-gphoto2", "gvfs-mtp", "gvfs-nfs", "gvfs-smb", "mousepad",
		"network-manager-applet", "parole", "ristretto", "sddm",
		"thunar-archive-plugin", "thunar-media-tags-plugin",
		"xdg-user-dirs-gtk", "xfce4", "xfce4-battery-plugin",
		"xfce4-mount-plugin", "xfce4-netload-plugin", "xfce4-notifyd",
		"xfce4-pulseaudio-plugin", "xfce4-screensaver",
		"xfce4-screenshooter", "xfce4-taskmanager", "xfce4-wavelan-plugin",
		"xfce4-weather-plugin", "xfce4-whiskermenu-plugin",
		"xfce4-xkb-plugin",
		"berserk-sddm-theme", "berserk-config-xfce",
	},
	"cinnamon": {
		"cinnamon", "cinnamon-translations", "file-roller", "galculator",
		"gnome-screenshot", "gnome-system-monitor", "gnome-terminal",
		"gthumb", "gvfs", "gvfs-afc", "gvfs-gphoto2", "gvfs-mtp",
		"gvfs-nfs", "gvfs-smb", "lightdm", "lightdm-slick-greeter", "mpv",
		"nemo-fileroller", "nemo-image-converter", "nemo-share", "x-apps",
		"xdg-user-dirs-gtk",
	},
	"hyprland": {
		"hyprland", "sddm", "waybar", "rofi-wayland", "swww", "xclip",
		"wl-clipboard", "xdotool", "qt5ct", "qt6ct", "kvantum",
		"xdg-desktop-portal-gtk", "xdg-desktop-portal-hyprland",
		"plasma-integration", "hyprland-qt-support", "hyprqt6engine",
		"hyprpolkitagent", "cliphist", "dunst", "nwg-look",
		"xdg-user-dirs-gtk",
		"berserk-sddm-theme", "berserk-user-config",
		"berserk-config-kde", "berserk-config-hyprland",
	},
	"i3": {
		"acpi", "arandr", "archlinux-xdg-menu", "awesome-terminal-fonts",
		"brightnessctl", "dex", "dmenu", "dunst", "feh", "galculator",
		"gvfs", "gvfs-afc", "gvfs-gphoto2", "gvfs-mtp", "gvfs-nfs",
		"gvfs-smb", "i3-wm", "i3blocks", "i3lock", "i3status", "jq",
		"lightdm", "lightdm-slick-greeter", "nwg-look", "mpv",
		"network-manager-applet", "numlockx", "playerctl", "polkit-gnome",
		"rofi", "scrot", "sysstat", "thunar", "thunar-archive-plugin",
		"thunar-volman", "tumbler", "unzip", "xarchiver", "xbindkeys",
		"xdg-user-dirs-gtk", "xed", "xfce4-terminal", "xorg-xdpyinfo",
		"xss-lock", "zip",
	},
}

// ByService maps a service/feature key to the packages that back it.
// Keys match ExtraServices map keys and the EnableX bool fields.
// Services that need no package (e.g. "trim") are simply absent.
var ByService = map[string][]string{
	"bluetooth": {"bluez", "bluez-utils"},
	"cups":      {"cups"},
	"ssh":       {"openssh"},
	"docker":    {"docker"},
	"tailscale": {"tailscale"},
	"avahi":     {"avahi"},
	"tlp":       {"tlp"},
	"ufw":       {"ufw"},
	"flatpak":   {"flatpak"},
	"snap":      {"snapd"},
	"zram":      {"zram-generator"},
}
