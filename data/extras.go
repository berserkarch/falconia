package data

// ExtraEntry is one row in the extra-packages picker — either a selectable
// package or a collapsible section header.
type ExtraEntry struct {
	Name     string
	Desc     string
	IsHeader bool
	Level    int // 0 = top-level header, 1 = item/sub-header, 2 = nested item
}

// ExtraCategory groups related entries under a named tab in the packages step.
type ExtraCategory struct {
	Name    string
	Entries []ExtraEntry
}

// ExtraCategories is the full catalog of user-selectable extra packages —
// the equivalent of Calamares netinstall.yaml.
// Edit here to add, remove, or reorganise the package picker.
var ExtraCategories = []ExtraCategory{
	{
		Name: "Internet",
		Entries: []ExtraEntry{
			{Name: "Browsers", IsHeader: true, Level: 0},
			{Name: "firefox", Desc: "Mozilla Firefox", Level: 1},
			{Name: "chromium", Desc: "Chromium open-source browser", Level: 1},
			{Name: "Communication", IsHeader: true, Level: 0},
			{Name: "discord", Desc: "Voice and text chat", Level: 1},
			{Name: "telegram-desktop", Desc: "Telegram Desktop client", Level: 1},
			{Name: "thunderbird", Desc: "Email and calendar client", Level: 1},
			{Name: "File Transfer", IsHeader: true, Level: 0},
			{Name: "transmission-qt", Desc: "BitTorrent client", Level: 1},
			{Name: "filezilla", Desc: "FTP/SFTP client", Level: 1},
		},
	},
	{
		Name: "Multimedia",
		Entries: []ExtraEntry{
			{Name: "Audio", IsHeader: true, Level: 0},
			{Name: "audacity", Desc: "Audio editor", Level: 1},
			{Name: "spotify-launcher", Desc: "Spotify music streaming", Level: 1},
			{Name: "Video", IsHeader: true, Level: 0},
			{Name: "vlc", Desc: "Media player", Level: 1},
			{Name: "mpv", Desc: "Lightweight media player", Level: 1},
			{Name: "kdenlive", Desc: "Video editor", Level: 1},
			{Name: "obs-studio", Desc: "Screen recorder and streaming", Level: 1},
			{Name: "Graphics", IsHeader: true, Level: 0},
			{Name: "gimp", Desc: "Image editor", Level: 1},
			{Name: "inkscape", Desc: "Vector graphics editor", Level: 1},
			{Name: "blender", Desc: "3D creation suite", Level: 1},
		},
	},
	{
		Name: "Dev Tools",
		Entries: []ExtraEntry{
			{Name: "Version Control", IsHeader: true, Level: 0},
			{Name: "git", Desc: "Version control system", Level: 1},
			{Name: "github-cli", Desc: "GitHub CLI", Level: 1},
			{Name: "Languages", IsHeader: true, Level: 0},
			{Name: "Python", IsHeader: true, Level: 1},
			{Name: "python", Desc: "Python 3 interpreter", Level: 2},
			{Name: "python-pip", Desc: "Python package installer", Level: 2},
			{Name: "Go", IsHeader: true, Level: 1},
			{Name: "go", Desc: "Go programming language", Level: 2},
			{Name: "Rust", IsHeader: true, Level: 1},
			{Name: "rustup", Desc: "Rust toolchain manager", Level: 2},
			{Name: "Node.js", IsHeader: true, Level: 1},
			{Name: "nodejs", Desc: "JavaScript runtime", Level: 2},
			{Name: "npm", Desc: "Node package manager", Level: 2},
			{Name: "Editors", IsHeader: true, Level: 0},
			{Name: "neovim", Desc: "Extensible terminal editor", Level: 1},
			{Name: "code", Desc: "Visual Studio Code", Level: 1},
			{Name: "Containers", IsHeader: true, Level: 0},
			{Name: "docker", Desc: "Container engine", Level: 1},
			{Name: "docker-compose", Desc: "Multi-container orchestration", Level: 1},
			{Name: "podman", Desc: "Daemonless container engine", Level: 1},
		},
	},
	{
		Name: "Office",
		Entries: []ExtraEntry{
			{Name: "Office Suite", IsHeader: true, Level: 0},
			{Name: "libreoffice-fresh", Desc: "Full-featured office suite", Level: 1},
			{Name: "libreoffice-still", Desc: "LibreOffice stable branch", Level: 1},
			{Name: "onlyoffice-desktopeditors", Desc: "OnlyOffice desktop", Level: 1},
			{Name: "PDF", IsHeader: true, Level: 0},
			{Name: "okular", Desc: "Document viewer", Level: 1},
			{Name: "evince", Desc: "GNOME document viewer", Level: 1},
			{Name: "Notes", IsHeader: true, Level: 0},
			{Name: "obsidian", Desc: "Markdown knowledge base", Level: 1},
			{Name: "joplin-desktop", Desc: "Note-taking app", Level: 1},
		},
	},
	{
		Name: "Security",
		Entries: []ExtraEntry{
			{Name: "Encryption", IsHeader: true, Level: 0},
			{Name: "gnupg", Desc: "GNU Privacy Guard", Level: 1},
			{Name: "age", Desc: "Simple file encryption", Level: 1},
			{Name: "Password Managers", IsHeader: true, Level: 0},
			{Name: "keepassxc", Desc: "Password manager", Level: 1},
			{Name: "pass", Desc: "Unix password manager", Level: 1},
			{Name: "Network Analysis", IsHeader: true, Level: 0},
			{Name: "nmap", Desc: "Network mapper", Level: 1},
			{Name: "wireshark-qt", Desc: "Network protocol analyzer", Level: 1},
			{Name: "Audit", IsHeader: true, Level: 0},
			{Name: "lynis", Desc: "Security auditing tool", Level: 1},
			{Name: "rkhunter", Desc: "Rootkit scanner", Level: 1},
		},
	},
	{
		Name: "Pentest",
		Entries: []ExtraEntry{
			{Name: "Information Gathering", IsHeader: true, Level: 0},
			{Name: "nmap", Desc: "Network mapper", Level: 1},
			{Name: "masscan", Desc: "Fast port scanner", Level: 1},
			{Name: "theharvester", Desc: "OSINT gathering tool", Level: 1},
			{Name: "maltego", Desc: "Link analysis tool", Level: 1},
			{Name: "Exploitation", IsHeader: true, Level: 0},
			{Name: "metasploit", Desc: "Penetration testing framework", Level: 1},
			{Name: "exploitdb", Desc: "Exploit database", Level: 1},
			{Name: "Password Attacks", IsHeader: true, Level: 0},
			{Name: "hashcat", Desc: "GPU password cracker", Level: 1},
			{Name: "john", Desc: "John the Ripper password cracker", Level: 1},
			{Name: "hydra", Desc: "Network login cracker", Level: 1},
			{Name: "Web", IsHeader: true, Level: 0},
			{Name: "burpsuite", Desc: "Web security testing platform", Level: 1},
			{Name: "sqlmap", Desc: "SQL injection tool", Level: 1},
			{Name: "nikto", Desc: "Web server scanner", Level: 1},
			{Name: "Wireless", IsHeader: true, Level: 0},
			{Name: "aircrack-ng", Desc: "WiFi security auditing", Level: 1},
			{Name: "wifite", Desc: "Automated WiFi auditor", Level: 1},
			{Name: "Forensics", IsHeader: true, Level: 0},
			{Name: "autopsy", Desc: "Digital forensics platform", Level: 1},
			{Name: "binwalk", Desc: "Firmware analysis tool", Level: 1},
			{Name: "volatility3", Desc: "Memory forensics framework", Level: 1},
		},
	},
	{
		Name: "System",
		Entries: []ExtraEntry{
			{Name: "Terminals", IsHeader: true, Level: 0},
			{Name: "kitty", Desc: "GPU-accelerated terminal", Level: 1},
			{Name: "alacritty", Desc: "Fast terminal emulator", Level: 1},
			{Name: "wezterm", Desc: "Multiplexing terminal emulator", Level: 1},
			{Name: "Shells", IsHeader: true, Level: 0},
			{Name: "fish", Desc: "Friendly interactive shell", Level: 1},
			{Name: "Printing", IsHeader: true, Level: 0},
			{Name: "cups", Desc: "Printing system", Level: 1},
			{Name: "hplip", Desc: "HP printer/scanner support", Level: 1},
			{Name: "Networking", IsHeader: true, Level: 0},
			{Name: "openssh", Desc: "SSH server and client", Level: 1},
			{Name: "tailscale", Desc: "Zero-config VPN", Level: 1},
			{Name: "firewalld", Desc: "Firewall management daemon", Level: 1},
			{Name: "Virtualisation", IsHeader: true, Level: 0},
			{Name: "virtualbox", Desc: "x86 virtualisation", Level: 1},
			{Name: "virt-manager", Desc: "KVM/QEMU virtual machine manager", Level: 1},
		},
	},
}
