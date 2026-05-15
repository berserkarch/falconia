# Falconia — Implementation Plan

## Vision

Falconia is a TUI installer for BerserkArch — an Arch Linux-based distro. It does not try to be a generic installer. It is Arch-specific, BerserkArch-opinionated, and ships as a single binary with zero external config files.

The reference for what a mature Arch installer does is the BerserkArch Calamares fork in `cal/`. Falconia takes the same installation logic but replaces the GUI with a TUI, replaces shell scripts and `.conf` modules with Go code, and replaces every external data file (package lists, service lists, driver mappings) with Go source files that compile into the binary.

---

## Core Philosophy

**1. Single binary, zero config files**
All data — package lists, DE packages, driver mappings, service lists — lives in `.go` files inside a `data/` package. No `.conf`, no `.yaml`, no external scripts. Everything compiles in.

**2. Arch-specific, not generic**
We rely directly on `parted`, `mkfs.*`, `pacstrap`, `arch-chroot`, `genfstab`, `dracut`, `cryptsetup`, `bootctl`, `grub-install`. No abstraction layers, no compatibility shims for other distros.

**3. Automatic where possible, explicit where it matters**
Hardware is detected silently at startup. The user never picks "install NVIDIA drivers" or "enable vboxservice" — the installer detects the hardware and handles it. The user makes choices that are genuinely theirs: disk, encryption, DE, kernel, bootloader, extra packages.

**4. Services are data, not code**
`data/services.go` has two flat lists: `Enable` and `Disable`. `EnableServices` iterates them unconditionally. If a service unit doesn't exist because the package wasn't installed, `systemctl enable` fails silently. No if-blocks, no flag checks.

**5. Dry-run always works**
Every installer function respects `cfg.DryRun`. The full Phase 2 pipeline must be walkable without root.

---

## Current State — What's Built

### Architecture
- `data/` package with `packages.go`, `services.go`, `extras.go`, `flow.go` — all static data as Go files
- `config.HardwareProfile` + `config.GPUInfo` structs; embedded in `InstallConfig`
- `installer/hardware.go` — `DetectHardware()` with CPU, GPU, WiFi, VM, RAM, NVMe probes
- `installer/drivers.go` — `InstallDrivers()` handling NVIDIA, Broadcom, VM guest tools
- Pipeline driven by `data.Pipeline` (equivalent to Calamares `settings.conf`); implementations registered in a registry map in `progress.go`

### TUI (Phase 1)
- Full step routing: Welcome → Disk → Partitions → Network → Locale → Hostname → Users → Kernel → Desktop → Packages → Bootloader → Confirm
- Sidebar showing live config summary throughout
- Advanced mode (`ctrl+a`) unlocks swap size on disk step, package descriptions on packages step
- Welcome screen shows detected hardware: CPU, GPU + driver decision, WiFi, RAM, NVMe, VM type
- Packages step reads categories from `data.ExtraCategories` (equivalent to Calamares `netinstall.yaml`)
- No services toggle screen — services are fully automatic

### Installer (Phase 2)
- Guided partitioning: UEFI (EFI + optional swap + root) and BIOS, NVMe/mmcblk suffix, `partprobe` + `udevadm settle`
- LUKS: `luksFormat`, `luksOpen`, keyfile generation + `luksAddKey`, dracut `crypt` module, `crypttab`; keyfile skipped for systemd-boot (ESP unencrypted)
- Filesystem: ext4, btrfs, xfs via `data.ByFilesystem`
- Plymouth: `add_dracutmodules+=" plymouth "`, `plymouth-set-default-theme berserk`, `quiet splash` in both bootloaders
- Microcode: auto-detected from `cfg.Hardware.CPU`, installed via `data.ByMicrocode`
- Kernel cmdline: `quiet splash` + `nvme_load=YES` (NVMe) + `nvidia-drm.modeset=1` (NVIDIA) — all hardware-driven
- GRUB: UEFI + BIOS, removable fallback, cryptodisk support, `grub-mkconfig`
- systemd-boot: kernel + initramfs + microcode copied to ESP, loader entry + `/etc/kernel/cmdline`
- Driver installation: NVIDIA (`nvidia` or `nvidia-open` based on device ID), hybrid GPU (`nvidia-prime`), Broadcom WiFi (`broadcom-wl` or `broadcom-wl-dkms` for LTS), VM guest tools, post-install dracut rebuild + grub-mkconfig
- Services: `data.Enable` tried unconditionally, `data.Disable` applied, `systemctl set-default graphical.target`
- Live file copy: `/etc/pacman.conf` from live ISO before pacstrap
- Copy live files step runs before pacstrap so custom repos are in place

---

## What's Left To Build

| Phase | Description | Status |
|---|---|---|
| B | Swap overhaul (none/partition/file/suspend modes) | Pending |
| E | Windows / dual-boot detection + boot entry | Pending |
| F | Post-install cleanup (machine ID, .pacnew, journal) | Pending |
| G | Btrfs subvolumes (`/@`, `/@home`, `/@cache`, `/@log`) | Pending (last) |

---

## Architecture

### Package layout (current state)

```
falconia/
├── main.go                      # DetectHardware() + DetectFirmware() before TUI starts
├── config/
│   └── config.go                # InstallConfig, User, HardwareProfile, GPUInfo
├── data/                        # All static data as Go files — the "config files"
│   ├── packages.go              # PackageList builder + Base, ByFilesystem, ByMicrocode,
│   │                            # CommonDE, ByDE, ByDriver, ByVM
│   ├── services.go              # Enable []string, Disable []string
│   ├── extras.go                # ExtraCategories — the TUI package picker catalog
│   └── flow.go                  # Pipeline []StepDef — ordered install steps + conditions
├── style/
│   └── styles.go                # Catppuccin Mocha palette + lipgloss styles
├── tui/
│   ├── app.go                   # Root bubbletea model, step routing, nav
│   ├── sidebar.go               # Live config summary panel (right side)
│   └── steps/
│       ├── messages.go          # Navigation message types
│       ├── welcome.go           # Step 00 — logo + hardware profile display
│       ├── disk.go              # Step 01 — disk, scheme, filesystem, encryption, swap
│       ├── partitions.go        # Step 01b — manual partition mapping
│       ├── network.go           # Step 02 — ethernet / WiFi / skip
│       ├── locale.go            # Step 03 — timezone, locale, keymap
│       ├── users.go             # Steps 04-05 — hostname, root password, user creation
│       ├── selections.go        # Generic radio list; kernel, desktop, bootloader steps
│       ├── packages.go          # Step 08 — categorised extra package selector
│       ├── confirm.go           # Step 09 — full summary + install button
│       └── progress.go          # Phase 2 — registry + pipeline runner + log viewport
└── installer/
    ├── runner.go                # Run, RunDry, RunChroot, RunChrootDry, runWithStdin
    ├── detect.go                # DetectFirmware, ListDisks, liveIsoDisk, CheckInternet
    ├── hardware.go              # DetectHardware — CPU, GPU, WiFi, VM, RAM, NVMe
    ├── disk.go                  # PartitionDisk, FormatDisks, MountDisks, Cleanup
    ├── base.go                  # Pacstrap, GenFstab, GenCrypttab
    ├── system.go                # SetTimezone, SetLocale, ConfigureNetwork, SetHostname, CreateUsers
    ├── bootloader.go            # InstallBootloader → installGrub / installSystemdBoot
    ├── drivers.go               # InstallDrivers → GPU, WiFi, VM tools
    ├── postinstall.go           # InstallDesktop, InstallPackages, EnableServices
    ├── livefiles.go             # CopyLiveFiles
    ├── helpers.go               # writeChroot, appendFile, runOutput
    ├── locales.go               # ListLocales
    └── timezone.go              # ListTimezones
```

---

## data/ Package — The "Config Files"

Each file corresponds to a Calamares `.conf` module:

| Calamares file | Falconia equivalent | Contents |
|---|---|---|
| `settings.conf` | `data/flow.go` | `Pipeline []StepDef` — ordered steps, labels, conditions |
| `services-systemd.conf` | `data/services.go` | `Enable []string`, `Disable []string` |
| `netinstall.yaml` | `data/extras.go` | `ExtraCategories` — TUI package picker catalog |
| `pacstrap.conf` / `packagechooser.conf` | `data/packages.go` | All package lists |

### `data/flow.go`

```go
var Pipeline = []StepDef{
    {StepVerifyInternet, "Verify internet", nil},
    {StepRankMirrors, "Rank mirrors", func(c *config.InstallConfig) bool { return c.RankMirrors }},
    {StepWriteCrypttab, "Write crypttab", func(c *config.InstallConfig) bool { return c.EncryptDisk }},
    {StepInstallDrivers, "Install hardware drivers", func(c *config.InstallConfig) bool {
        // only runs when there's hardware that needs a driver
    }},
    // ...
}
```

Adding a step: add a `StepKey` constant + `StepDef` entry here, then register the implementation in `progress.go`.

### `data/services.go`

```go
var Enable = []string{
    "NetworkManager", "systemd-timesyncd", "fstrim.timer", "sddm", "gdm",
    "lightdm", "docker", "sshd", "nvidia-persistenced", "vboxservice", ...
}
var Disable = []string{"bluetooth", "pacman-init"}
```

Always tried unconditionally. Failures are warnings.

### `data/packages.go`

```go
var Base = []string{ ... }                      // every install
var CommonDE = []string{ ... }                  // any DE/WM selection
var ByDE = map[string][]string{ "kde": {...} }  // per-DE packages
var ByDriver = map[string][]string{ ... }       // nvidia, nvidia-open, broadcom, broadcom-lts
var ByVM = map[string][]string{ ... }           // virtualbox, vmware, kvm, qemu
var ByMicrocode = map[string][]string{ ... }    // intel, amd
var ByFilesystem = map[string][]string{ ... }   // ext4, btrfs, xfs
```

PackageList builder:
```go
pkgs := data.New().
    Add(data.Base...).
    Add(cfg.Kernel, cfg.Kernel+"-headers").
    AddMap(data.ByFilesystem, cfg.Filesystem).
    AddIf(cfg.EncryptDisk, data.Encryption...).
    AddMap(data.ByMicrocode, cfg.Hardware.CPU).
    Build()  // deduplicates
```

---

## TUI — Phase 1 Steps (current)

| # | File | What the user configures |
|---|---|---|
| 00 | `welcome.go` | Logo + **full hardware profile** (CPU, GPU + driver, WiFi, RAM, NVMe, VM) |
| 01 | `disk.go` | Target disk, guided/manual, filesystem, encryption + passphrase, swap size |
| 01b | `partitions.go` | Manual only — maps `/`, `/boot/efi`, `swap` to partitions |
| 02 | `network.go` | Ethernet / WiFi / skip |
| 03 | `locale.go` | Timezone (searchable), locale, keymap |
| 04 | `users.go` | Hostname |
| 05 | `users.go` | Root password, non-root user (name, password, shell, groups) |
| 06 | `selections.go` | Kernel variant |
| 07 | `selections.go` | Desktop environment (none/kde/gnome/xfce/cinnamon/hyprland/i3) |
| 08 | `packages.go` | Extra packages by category (from `data.ExtraCategories`) |
| 09 | `selections.go` | Bootloader |
| 10 | `confirm.go` | Read-only summary; **Install** button triggers Phase 2 |

No services toggle step — services are data-driven and fully automatic.

---

## Install Pipeline — Phase 2 (current)

Driven by `data.Pipeline`. Implementations registered in `stepRegistry` in `progress.go`.

| # | Key | Label | Condition |
|---|---|---|---|
| 1 | `verify-internet` | Verify internet | always |
| 2 | `sync-clock` | Sync system clock | always |
| 3 | `rank-mirrors` | Rank mirrors | `cfg.RankMirrors` |
| 4 | `partition-disk` | Partition disk | always |
| 5 | `format-disks` | Format filesystems | always |
| 6 | `mount-disks` | Mount filesystems | always |
| 7 | `copy-live-files` | Copy live environment files | always |
| 8 | `pacstrap` | Install base system | always |
| 9 | `gen-fstab` | Generate fstab | always |
| 10 | `set-timezone` | Set timezone | always |
| 11 | `write-crypttab` | Write crypttab | `cfg.EncryptDisk` |
| 12 | `set-locale` | Set locale | always |
| 13 | `config-network` | Configure network | always |
| 14 | `set-hostname` | Set hostname | always |
| 15 | `root-password` | Set root password | always |
| 16 | `create-users` | Create users | always |
| 17 | `bootloader` | Install bootloader | always |
| 18 | `install-desktop` | Install desktop environment | `cfg.DesktopEnv != "none"` |
| 19 | `install-packages` | Install extra packages | always |
| 20 | `install-drivers` | Install hardware drivers | NVIDIA/Broadcom/VM detected |
| 21 | `enable-services` | Enable services | always |
| 22 | `cleanup` | Unmount & cleanup | always |

---

## Remaining Phases

### Phase B — Swap Overhaul
Add `SwapMode` type (`none` / `partition` / `file` / `suspend`) to `config/config.go`. Update disk TUI step to show a mode picker instead of just a size field. Update `PartitionDisk` to skip the swap partition for file/none/suspend modes. Implement swap file creation post-mount. For suspend mode: auto-size to RAM, add `resume=` to kernel cmdline. Handle btrfs + swap file (`chattr +C` before `mkswap`). Update `rootPartition()` to key off `SwapMode == SwapPartition` instead of `SwapSize > 0`.

### Phase E — Windows / Dual-Boot
Add `os-prober` to `data.Base`. Write `DetectWindows()` in `installer/detect.go` — runs post-partitioning, returns the Windows partition if found. Add `StepDetectWindows` to pipeline. For GRUB: set `GRUB_DISABLE_OS_PROBER=false` (grub-mkconfig handles the rest). For systemd-boot: write `/boot/efi/loader/entries/windows.conf` manually.

### Phase F — Post-Install Cleanup
Write `PostInstallCleanup()` in `installer/postinstall.go` covering:
- `systemd-machine-id-setup` — generate machine ID
- Remove wrong-CPU microcode package (`pacman -R amd-ucode` on Intel, vice versa)
- `find /mnt -name "*.pacnew" -delete`
- Set journal storage to persistent (`Storage=auto` in `journald.conf`)

Add `StepPostCleanup` to `data/flow.go` before `StepCleanup`.

### Phase G — Btrfs Subvolumes (last)
When `cfg.Filesystem == "btrfs"`, `FormatDisks` should create named subvolumes after `mkfs.btrfs`: mount root, `btrfs subvolume create /@`, `/@home`, `/@cache`, `/@log`, unmount, then remount at `subvol=/@`. `MountDisks` binds each subvolume. `genfstab` captures them. Handle swap file on btrfs (`chattr +C` + offset calculation for suspend).

---

## Open Questions

- **EFI mount point:** Currently `/boot/efi`. Calamares uses `/efi`. No change planned — `/boot/efi` is correct for our GRUB setup (kernel inside encrypted /boot, only GRUB EFI binary on ESP).
- **Offline install:** Out of scope. Online (pacstrap) only.
- **GeoIP timezone:** Nice-to-have for the locale step. Not blocking.
- **NVIDIA UseOpen heuristic:** Turing+ (device ID ≥ `0x1e00`) → `nvidia-open`. Approximate but correct for the vast majority of consumer GPUs. Worst case: proprietary nvidia works on all supported cards.
- **Surface / marvell firmware:** Narrow edge case, skip for now.
