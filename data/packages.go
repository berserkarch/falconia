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
	"os-prober",
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

// CommonDE is installed for any desktop environment or window manager selection.
// These are the shared foundation — xorg, audio, kitty, and desktop integration
// essentials that every DE/WM expects to be present.
var CommonDE = []string{
	// Unified terminal across all DEs
	"kitty",
	// X11 base (used directly by X11 DEs; Wayland DEs use it via XWayland)
	"extra/xorg",
	"extra/xorg-server",
	"extra/xorg-xinit",
	"extra/xorg-xrandr",
	"extra/xorg-xdpyinfo",
	"extra/xorg-xinput",
	"extra/xorg-xkill",
	"xf86-input-libinput",
	// Open-source GPU rendering
	"mesa",
	"libva-mesa-driver",
	// Audio (PipeWire replaces PulseAudio/JACK)
	"pipewire",
	"pipewire-pulse",
	"pipewire-alsa",
	"jack2",
	"wireplumber",
	// Desktop integration
	"xdg-user-dirs",
	"xdg-utils",
	"polkit",
	"gvfs",
	"ttf-jetbrains-mono-nerd",
	"ttf-firacode-nerd",
	"visual-studio-code-bin",
	"sublime-text-4",
	"duf",
	"fd",
	"fzf",
	"ripgrep",
	"go",
	"findutils",
	"git",
	"glances",
	"hwinfo",
	"inxi",
	"nano-syntax-highlighting",
	"python-defusedxml",
	"python-jinja",
	"python-packaging",
	"rsync",
	"tldr",
	"sed",
	"vi",
	"vim",
	"wget",
	"pamac",
	"exa",
	"p7zip",
	"zip",
	"apparmor",
	"audit",
	"firewalld",
	"docker",
	"unarchiver",
	"dmidecode",
	"dmraid",
	"hdparm",
	"hwdetect",
	"lsscsi",
	"mtools",
	"sg3_utils",
	"sof-firmware",
	"cantarell-fonts",
	"freetype2",
	"noto-fonts",
	"noto-fonts-emoji",
	"noto-fonts-cjk",
	"noto-fonts-extra",
	"ttf-bitstream-vera",
	"ttf-dejavu",
	"ttf-liberation",
	"ttf-opensans",
	"adobe-source-code-pro-fonts",
	"terminus-font",
	"unrar",
	"unzip",
	"xz",
	"downgrade",
	"pacman-contrib",
	"pkgfile",
	"rebuild-detector",
	"reflector",
	"b43-fwcutter",
	"dialog",
	"dnsmasq",
	"dnsutils",
	"ethtool",
	"iwd",
	"modemmanager",
	"networkmanager",
	"networkmanager-openconnect",
	"networkmanager-openvpn",
	"openvpn",
	"nss-mdns",
	"openssh",
	"usb_modeswitch",
	"whois",
	"wireless-regdb",
	"xl2tpd",
	"wpa_supplicant",
	"yay",
	"qt5ct",
	"qt6ct",
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
		"berserk-config-gnome", "berserk-sddm-theme",
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
		"xfce4-xkb-plugin", "berserk-sddm-theme", "berserk-config-xfce",
	},
	// "cinnamon": {
	// 	"cinnamon", "cinnamon-translations", "file-roller", "galculator",
	// 	"gnome-screenshot", "gnome-system-monitor", "gnome-terminal",
	// 	"gthumb", "gvfs", "gvfs-afc", "gvfs-gphoto2", "gvfs-mtp",
	// 	"gvfs-nfs", "gvfs-smb", "sddm", "berserk-sddm-theme", "mpv",
	// 	"nemo-fileroller", "nemo-image-converter", "nemo-share", "x-apps",
	// 	"xdg-user-dirs-gtk",
	// },
	"hyprland": {
		"hyprland", "sddm", "waybar", "rofi-wayland", "swww", "xclip",
		"wl-clipboard", "xdotool", "qt5ct", "qt6ct", "kvantum",
		"xdg-desktop-portal-gtk", "xdg-desktop-portal-hyprland",
		"plasma-integration", "hyprland-qt-support", "hyprqt6engine",
		"hyprpolkitagent", "cliphist", "dunst", "nwg-look",
		"xdg-user-dirs-gtk", "berserk-sddm-theme", "berserk-user-config",
		"berserk-config-kde", "berserk-config-hyprland",
	},
	"i3": {
		"acpi", "arandr", "archlinux-xdg-menu", "awesome-terminal-fonts",
		"brightnessctl", "dex", "dmenu", "dunst", "feh", "gvfs", "gvfs-afc",
		"gvfs-gphoto2", "gvfs-mtp", "gvfs-nfs", "gvfs-smb", "i3-wm", "i3blocks",
		"i3lock", "i3status", "jq", "sddm", "berserk-sddm-theme", "nwg-look", "mpv",
		"network-manager-applet", "numlockx", "playerctl", "polkit-kde-agent",
		"rofi", "scrot", "sysstat", "thunar", "thunar-archive-plugin", "berserk-config-i3wm",
		"thunar-volman", "tumbler", "unzip", "xarchiver", "xbindkeys", "polybar",
		"xdg-user-dirs-gtk", "xed", "xorg-xdpyinfo", "xss-lock", "zip",
	},
	"dwm": {
		"acpi", "arandr", "archlinux-xdg-menu", "awesome-terminal-fonts",
		"brightnessctl", "dex", "dmenu", "dunst", "feh", "gvfs", "gvfs-afc",
		"gvfs-gphoto2", "gvfs-mtp", "gvfs-nfs", "gvfs-smb", "jq", "sddm", "berserk-sddm-theme",
		"nwg-look", "mpv", "network-manager-applet", "numlockx", "playerctl", "polkit-kde-agent",
		"rofi", "scrot", "sysstat", "thunar", "thunar-archive-plugin", "berserk-config-dwm",
		"thunar-volman", "tumbler", "unzip", "xarchiver", "xbindkeys", "xdg-user-dirs-gtk", "xed", "xorg-xdpyinfo", "zip",
	},
	"openbox": {
		"obmenu-generator", "berserk-config-openbox", "sddm", "berserk-sddm-theme",
		"thunar",
	},
}

// ByDriver maps a GPU driver scenario to its packages.
// "nvidia" = proprietary, "nvidia-open" = open kernel module (Turing+).
// AMD and Intel drivers ship with the kernel — no extra package needed.
var ByDriver = map[string][]string{
	"nvidia":       {"nvidia", "nvidia-utils", "nvidia-settings"},
	"nvidia-open":  {"nvidia-open", "nvidia-utils", "nvidia-settings"},
	"broadcom":     {"broadcom-wl"},
	"broadcom-lts": {"broadcom-wl-dkms"},
}

// ByVM maps a detected virtualisation platform to its guest tool packages.
var ByVM = map[string][]string{
	"virtualbox": {"virtualbox-guest-utils"},
	"vmware":     {"open-vm-tools", "gtkmm3", "xf86-input-vmmouse"},
	"kvm":        {"qemu-guest-agent", "spice-vdagent"},
	"qemu":       {"qemu-guest-agent", "spice-vdagent"},
}
