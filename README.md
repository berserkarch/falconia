# Falconia

Official TUI installer for [BerserkArch](https://berserkarch.org) — an Arch Linux-based distribution.

Falconia ships as a single binary with no external config files. All package lists, service definitions, and pipeline logic compile into the binary.

---

## Features

- Guided and manual partitioning (UEFI GPT and BIOS MBR)
- Filesystems: ext4, btrfs, xfs
- Full-disk LUKS encryption with keyfile (eliminates double passphrase prompt on GRUB)
- Desktop environments: GNOME, KDE Plasma, XFCE, Cinnamon, Hyprland, i3, Openbox, or none
- Kernels: linux, linux-lts, linux-zen, linux-hardened
- Bootloaders: GRUB (UEFI + BIOS) and systemd-boot (UEFI only)
- Automatic hardware detection: CPU microcode, NVIDIA (open or proprietary), Broadcom WiFi, VM guest tools
- Windows dual-boot detection and boot entry generation
- Dry-run mode — walk the full install pipeline without root or a real disk

---

## Usage

Boot into a BerserkArch live ISO, then run:

```bash
sudo falconia
```

For testing without a disk:

```bash
falconia --dry-run
```

### Navigation

| Key | Action |
|---|---|
| `↑↓` / `tab` | Move between fields |
| `←→` | Change value / cycle options |
| `enter` | Confirm / advance |
| `esc` | Go back |
| `ctrl+a` | Toggle advanced mode |
| `q` | Quit |

Advanced mode unlocks swap size control (disk step) and package descriptions (packages step).

---

## Install flow

**Phase 1 — Configuration**

| Step | What you configure |
|---|---|
| Welcome | Detected hardware summary |
| Disk | Target disk, partitioning scheme, filesystem, encryption, swap |
| Network | Ethernet / WiFi / skip |
| Locale | Timezone, locale, keymap |
| Hostname | Machine name |
| Users | Root password, user account |
| Kernel | Kernel variant |
| Desktop | Desktop environment |
| Packages | Extra packages by category |
| Bootloader | GRUB or systemd-boot |
| Confirm | Full summary → Install |

**Phase 2 — Installation**

Runs sequentially with live log output. Steps include pacstrap, dracut initramfs generation, bootloader installation, driver installation, service enablement, and cleanup.

---

## Building

Requires Go 1.21+.

```bash
go build -o falconia .
```

---

## Architecture

```
falconia/
├── main.go            # Hardware detection, TUI startup
├── config/            # InstallConfig — single source of truth
├── data/              # All static data as Go variables (packages, services, pipeline)
├── installer/         # Phase 2 implementation functions
├── tui/               # Bubbletea models (app router, sidebar, steps)
└── style/             # Catppuccin Mocha palette and lipgloss styles
```

`data/` is the equivalent of Calamares `.conf` files — package lists, service lists, and the ordered install pipeline all live here as Go source and compile into the binary.

---

## License

MIT
