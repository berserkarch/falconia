# Falconia — Developer Guide

Falconia is a Go TUI installer for BerserkArch (an Arch Linux-based distro). It walks the user through configuration in Phase 1, then executes the installation pipeline sequentially in Phase 2, streaming live output to the screen.

---

## Building & Running

```bash
go build -o falconia .

# Run as root (required for real installs)
sudo ./falconia

# Dry-run mode — no root needed, commands are logged but not executed
./falconia --dry-run
```

---

## Project Layout

```
falconia/
├── main.go                      # Entry point — hardware/firmware detection, TUI launch
├── config/
│   └── config.go                # InstallConfig, User, HardwareProfile, GPUInfo structs
├── data/                        # All static data as Go files (no external config files)
│   ├── packages.go              # PackageList builder + all package lists
│   ├── services.go              # Enable/Disable service lists
│   ├── extras.go                # ExtraCategories — TUI package picker catalog
│   └── flow.go                  # Pipeline — ordered install step definitions
├── style/
│   └── styles.go                # Catppuccin Mocha palette, lipgloss styles, helpers
├── tui/
│   ├── app.go                   # Root bubbletea model, step routing, nav
│   ├── sidebar.go               # Live config summary panel (right side)
│   └── steps/
│       ├── messages.go          # Navigation message types (Done, Back, StartInstall)
│       ├── welcome.go           # Step 00 — logo + hardware profile display
│       ├── disk.go              # Step 01 — disk, scheme, filesystem, encryption, swap
│       ├── partitions.go        # Step 01b — manual partition mapping
│       ├── network.go           # Step 02 — ethernet / WiFi / skip
│       ├── locale.go            # Step 03 — timezone, locale, keymap
│       ├── users.go             # Steps 04-05 — hostname, root password, user creation
│       ├── selections.go        # Generic radio list; kernel, desktop, bootloader steps
│       ├── packages.go          # Step 08 — categorised extra package selector
│       ├── confirm.go           # Step 09 — full summary + install button
│       └── progress.go          # Phase 2 — step registry, pipeline runner, log viewport
└── installer/
    ├── runner.go                # Run, RunDry, RunChroot, RunChrootDry, runWithStdin
    ├── detect.go                # DetectFirmware, ListDisks, CheckInternet, DetectWindows
    ├── hardware.go              # DetectHardware — CPU, GPU, WiFi, VM, RAM, NVMe
    ├── disk.go                  # PartitionDisk, FormatDisks, MountDisks, Cleanup
    ├── base.go                  # Pacstrap, GenFstab, GenCrypttab
    ├── system.go                # SetTimezone, SetLocale, ConfigureNetwork, SetHostname, CreateUsers
    ├── bootloader.go            # InstallBootloader → installGrub / installSystemdBoot
    ├── drivers.go               # InstallDrivers → GPU, WiFi, VM guest tools
    ├── postinstall.go           # InstallDesktop, InstallPackages, EnableServices, PostInstallCleanup
    ├── livefiles.go             # CopyLiveFiles
    ├── helpers.go               # writeChroot, appendFile, runOutput
    ├── locales.go               # ListLocales
    └── timezone.go              # ListTimezones
```

---

## config.InstallConfig

Single struct passed to every installer function. Set during Phase 1, consumed during Phase 2.

| Field | Type | Description |
|---|---|---|
| `Hardware` | `HardwareProfile` | Auto-detected at startup — never user-set |
| `Firmware` | string | `"uefi"` or `"bios"` — auto-detected |
| `Disk` | string | Target block device e.g. `/dev/sda` |
| `DiskModel` | string | Human label shown on confirm screen |
| `PartitionScheme` | string | `"guided"` or `"manual"` |
| `Filesystem` | string | `"ext4"`, `"btrfs"`, or `"xfs"` |
| `SwapSize` | int | Swap partition size in MiB; `0` = none |
| `EncryptDisk` | bool | Enable LUKS encryption on root partition |
| `EncryptionPass` | string | LUKS passphrase — never logged |
| `NetworkMode` | string | `"wifi"`, `"ethernet"`, or `"skip"` |
| `WifiSSID` | string | WiFi network name |
| `WifiPass` | string | WiFi password — never logged |
| `Timezone` | string | e.g. `"Asia/Kolkata"` |
| `Locale` | string | e.g. `"en_US.UTF-8"` |
| `Keymap` | string | e.g. `"us"` |
| `Hostname` | string | System hostname |
| `RootPassword` | string | Root password — never logged |
| `Users` | `[]User` | Non-root users (username, password, shell, groups) |
| `Kernel` | string | `"linux"`, `"linux-lts"`, `"linux-zen"`, `"linux-hardened"` |
| `DesktopEnv` | string | `"none"`, `"kde"`, `"gnome"`, `"xfce"`, `"cinnamon"`, `"hyprland"`, `"i3"` |
| `DisplayManager` | string | Auto-set from DE choice via `DEDisplayManager()` |
| `ExtraPackages` | `[]string` | User-selected packages from the packages step |
| `Bootloader` | string | `"grub"` or `"systemd-boot"` |
| `GrubTimeout` | int | Bootloader menu timeout in seconds |
| `RankMirrors` | bool | Run reflector before pacstrap |
| `WindowsEFIPath` | string | Non-empty when a Windows install is detected; set by `DetectWindows()` |
| `MountPoints` | `map[string]string` | Manual partitioning: mount point → device |
| `DryRun` | bool | Skip actual command execution |

### HardwareProfile

Populated once at startup by `installer.DetectHardware()`. Never changes after that.

| Field | Type | Values |
|---|---|---|
| `CPU` | string | `"intel"` \| `"amd"` \| `"other"` |
| `GPUs` | `[]GPUInfo` | One entry per detected display adapter |
| `WiFi` | string | `"broadcom"` \| `"other"` |
| `VM` | string | `"virtualbox"` \| `"vmware"` \| `"kvm"` \| `"qemu"` \| `"none"` |
| `RAMBytes` | int64 | Total physical RAM in bytes |
| `HasNVMe` | bool | Any `/dev/nvme*` device present |

`GPUInfo.UseOpen` is `true` for NVIDIA Turing+ (device ID ≥ `0x1e00`) — installs `nvidia-open` instead of `nvidia`.

---

## data/ Package

The `data/` package is the equivalent of Calamares `.conf` files — pure data, no logic.

### `data/flow.go` — Pipeline (settings.conf equivalent)

Defines the ordered install sequence. Each `StepDef` has a typed `StepKey`, a display label, an optional condition function, and a `Soft bool`.

`Soft: true` steps log a warning on failure and continue — they do not abort the installation. Use this for steps where failure is non-fatal (e.g. clock sync, mirror ranking, OS detection).

```go
var Pipeline = []StepDef{
    {Key: StepVerifyInternet, Label: "Verify internet"},
    {Key: StepSyncClock,      Label: "Sync system clock", Soft: true},
    {Key: StepRankMirrors,    Label: "Rank mirrors", Soft: true, When: func(c *InstallConfig) bool {
        return c.RankMirrors
    }},
    // ...
}
```

To add a new step:
1. Declare a `StepKey` constant in `data/flow.go`
2. Add a `StepDef` to `Pipeline` at the right position (with `Soft: true` if appropriate)
3. Register the implementation function in `stepRegistry` in `tui/steps/progress.go`

### `data/services.go` — Services (services-systemd.conf equivalent)

Two flat lists. `EnableServices` iterates them unconditionally — failures are logged as warnings and execution continues.

```go
var Enable = []string{
    "NetworkManager", "systemd-timesyncd", "fstrim.timer",
    "sddm", "gdm", "lightdm",           // only the installed DM succeeds
    "docker", "sshd", "cups",            // succeed if package was selected
    "nvidia-persistenced",               // succeeds on NVIDIA installs
    "vboxservice", "vmtoolsd", ...       // succeed on matching VM
}
var Disable = []string{"bluetooth", "pacman-init"}
```

### `data/extras.go` — Extra package catalog (netinstall.yaml equivalent)

Defines the categories and entries shown in the packages TUI step. The TUI reads `data.ExtraCategories` and converts them to its internal `pkgCategory` / `pkgEntry` types (which also carry runtime state like `checked` and `collapsed`).

```go
var ExtraCategories = []ExtraCategory{
    {Name: "Internet", Entries: []ExtraEntry{
        {Name: "Browsers", IsHeader: true, Level: 0},
        {Name: "firefox", Desc: "Mozilla Firefox", Level: 1},
        // ...
    }},
    // ...
}
```

To add a package to the picker: one line in `data/extras.go`.

### `data/packages.go` — Package lists (pacstrap.conf / packagechooser.conf equivalent)

All package lists as exported variables. The `PackageList` builder assembles them conditionally:

```go
pkgs := data.New().
    Add(data.Base...).
    Add(cfg.Kernel, cfg.Kernel+"-headers").
    AddMap(data.ByFilesystem, cfg.Filesystem).   // looks up "ext4", "btrfs", or "xfs"
    AddIf(cfg.EncryptDisk, data.Encryption...).
    AddMap(data.ByMicrocode, cfg.Hardware.CPU).  // looks up "intel" or "amd"
    Build()                                       // deduplicates, preserves order
```

Key variables:

| Variable | Used for |
|---|---|
| `Base` | Every install via pacstrap |
| `CommonDE` | Any DE/WM selection (xorg, kitty, mesa, pipewire, etc.) |
| `ByDE` | Per-DE packages sourced from Calamares `packagechooser.conf` |
| `ByFilesystem` | e2fsprogs / btrfs-progs / xfsprogs |
| `ByMicrocode` | intel-ucode / amd-ucode |
| `ByDriver` | nvidia, nvidia-open, broadcom, broadcom-lts |
| `ByVM` | virtualbox-guest-utils, open-vm-tools, qemu-guest-agent |
| `Encryption` | cryptsetup |

---

## Phase 1 — TUI Configuration Steps

| # | File | What the user configures |
|---|---|---|
| 00 | `welcome.go` | Logo + **hardware profile** — CPU + ucode, GPU + driver, WiFi, RAM, NVMe, VM |
| 01 | `disk.go` | Target disk, guided/manual, filesystem, encryption + passphrase, swap size |
| 01b | `partitions.go` | Manual only — maps `/`, `/boot/efi`, `swap` to specific partitions |
| 02 | `network.go` | Ethernet / WiFi (SSID + password) / skip |
| 03 | `locale.go` | Timezone (searchable list), locale, keymap |
| 04 | `users.go` | Hostname |
| 05 | `users.go` | Root password, non-root user (name, password, shell, groups) |
| 06 | `selections.go` | Kernel variant |
| 07 | `selections.go` | Desktop environment |
| 08 | `packages.go` | Extra packages by category with collapsible headers |
| 09 | `selections.go` | Bootloader |
| 10 | `confirm.go` | Read-only summary; **Install** button triggers Phase 2 |

There is no services toggle step. Services are data-driven and fully automatic.

**Key nav:** `↑↓` / `tab` move between fields; `←→` change values; `enter` advances; `esc` goes back; `ctrl+a` toggles advanced mode.

---

## Phase 2 — Install Pipeline

Driven by `data.Pipeline`. `buildSteps()` in `progress.go` iterates `data.Pipeline`, filters by condition, and looks up each step's implementation in `stepRegistry`. Adding a step requires touching three places: `data/flow.go` (key + definition), `progress.go` (registry entry), and `installer/` (implementation function).

**Soft steps** log a yellow warning on failure and continue — the install does not abort.

| # | Label | Condition | Soft |
|---|---|---|---|
| 1 | Verify internet | always | — |
| 2 | Sync system clock | always | ✓ |
| 3 | Rank mirrors | `cfg.RankMirrors` | ✓ |
| 4 | Partition disk | always | — |
| 5 | Format filesystems | always | — |
| 6 | Mount filesystems | always | — |
| 7 | Copy live environment files | always | — |
| 8 | Install base system (pacstrap) | always | — |
| 9 | Generate fstab | always | — |
| 10 | Set timezone | always | — |
| 11 | Write crypttab | `cfg.EncryptDisk` | — |
| 12 | Set locale | always | — |
| 13 | Configure network | always | — |
| 14 | Set hostname | always | — |
| 15 | Set root password | always | — |
| 16 | Create users | always | — |
| 17 | Detect other OS | always | ✓ |
| 18 | Install bootloader | always | — |
| 19 | Install desktop environment | `cfg.DesktopEnv != "none"` | — |
| 20 | Install extra packages | always | — |
| 21 | Install hardware drivers | NVIDIA or Broadcom or VM detected | — |
| 22 | Enable services | always | — |
| 23 | Post-install cleanup | always | — |
| 24 | Unmount & cleanup | always | — |

---

## installer/ Package

### runner.go — Command execution

```go
Run(log, "pacman", "-S", "vim")               // run on host
RunDry(cfg, log, "pacman", "-S", "vim")       // skips if cfg.DryRun
RunChroot(log, "pacman", "-S", "vim")         // arch-chroot /mnt
RunChrootDry(cfg, log, "pacman", "-S", "vim") // arch-chroot, DryRun-aware
runWithStdin(log, reader, "cryptsetup", ...)  // piped stdin (passwords)
runOutput("blkid", "-s", "UUID", ...)         // capture stdout
```

### hardware.go — Detection

`DetectHardware()` runs once in `main.go` before the TUI starts. It probes:
- **CPU** — `/proc/cpuinfo` for `GenuineIntel` / `AuthenticAMD`
- **GPUs** — `lspci -nn` for class `0300/0302/0380`; PCI vendor `10de`=NVIDIA, `1002`=AMD, `8086`=Intel; device ID ≥ `0x1e00` → `UseOpen=true`
- **WiFi** — `lspci -nn` for vendor `14e4` (Broadcom)
- **VM** — `systemd-detect-virt`; maps `oracle`→`virtualbox`
- **RAM** — `/proc/meminfo` MemTotal in bytes
- **NVMe** — any `/dev/nvme*` in `/dev`

### disk.go — Partitioning

**Guided layout:**

UEFI: `sda1` EFI 512 MiB (FAT32) · `sda2` swap (if SwapSize > 0) · `sda3` root
BIOS: `sda1` BIOS-boot 1 MiB · `sda2` swap (if SwapSize > 0) · `sda3` root

After `parted`, `partprobe <disk> && udevadm settle` forces the kernel to see the new layout before `mkfs` runs (prevents mount-of-old-LUKS-partition bug).

**LUKS:** `cryptsetup luksFormat --type luks1` → `cryptsetup open` → `mkfs` on `/dev/mapper/cryptroot`.

### base.go — Pacstrap + initramfs

Beyond running pacstrap, `Pacstrap()` also:
1. Writes `/mnt/etc/dracut.conf.d/plymouth.conf` — embeds Plymouth in initramfs
2. If encrypted + GRUB: writes `encryption.conf`, generates 512-byte keyfile at `/mnt/crypto_keyfile.bin` (mode `0000`), registers it via `luksAddKey` (no second passphrase prompt at boot)
3. If encrypted + systemd-boot: no keyfile (ESP is unencrypted; initramfs prompts once)
4. `plymouth-set-default-theme berserk` in chroot before dracut
5. `dracut --force` (primary) + `dracut --no-hostonly --force` (fallback)

### bootloader.go — GRUB & systemd-boot

Kernel cmdline is assembled from hardware detection — not hardcoded:
- `quiet splash` — always
- `nvme_load=YES` — when `cfg.Hardware.HasNVMe`
- `nvidia-drm.modeset=1` — when any `cfg.Hardware.GPUs` has `Vendor == "nvidia"`
- `rd.luks.uuid=...` + `rd.luks.key=...` — when encrypted + GRUB
- `rd.luks.uuid=...` (no key) — when encrypted + systemd-boot

GRUB also runs `grub-install --removable` as an NVRAM-fallback for firmware that ignores boot entries.

systemd-boot writes `/mnt/etc/kernel/cmdline` so future `kernel-install` invocations (triggered by kernel package upgrades) preserve the correct parameters.

**Dual-boot:** when `cfg.WindowsEFIPath != ""` (set by the `detect-windows` step):
- GRUB: adds `GRUB_DISABLE_OS_PROBER=false` to `/etc/default/grub` before `grub-mkconfig`; `grub-mkconfig` picks up Windows automatically via `os-prober`
- systemd-boot: writes `/boot/efi/loader/entries/windows.conf` pointing to `/EFI/Microsoft/Boot/bootmgfw.efi`

### drivers.go — Hardware driver installation

`InstallDrivers()` is the Phase 2 step for hardware-specific packages:

- **NVIDIA** — installs `data.ByDriver["nvidia"]` or `["nvidia-open"]` based on `gpu.UseOpen`; adds `nvidia-prime` for hybrid Intel+NVIDIA; rebuilds dracut initramfs; regenerates grub.cfg
- **Broadcom WiFi** — installs `broadcom-wl` (normal kernels) or `broadcom-wl-dkms` (`linux-lts`)
- **VM guest tools** — installs from `data.ByVM[cfg.Hardware.VM]`; removes `power-profiles-daemon` (irrelevant in VMs)

### postinstall.go — Desktop, packages, services, cleanup

**`InstallDesktop`** — installs `data.CommonDE` (shared xorg/audio/kitty foundation) + `data.ByDE[cfg.DesktopEnv]` (DE-specific), deduplicated.

**`InstallPackages`** — installs only `cfg.ExtraPackages` (what the user picked in the packages step). No flag-based additions.

**`EnableServices`** — iterates `data.Enable`, then `data.Disable`, then sets `graphical.target`. No conditionals anywhere.

**`PostInstallCleanup`** — final housekeeping, always runs before unmounting:
- `systemd-machine-id-setup` in chroot — replaces the live ISO's machine ID with a fresh one
- `find /mnt -name "*.pacnew" -delete` — removes config remnants from pacstrap
- Writes `/etc/systemd/journald.conf.d/00-persistence.conf` with `Storage=persistent` — journal survives reboots

### livefiles.go — Live ISO file copy

```go
var liveFiles = []string{
    "/etc/pacman.conf",
    // add more paths here
}
```

Runs before pacstrap so the BerserkArch repo config is in place when packages are installed. Add entries here as needed.

### detect.go — Disk and OS detection

`ListDisks()` excludes:
- Loop devices + optical drives (`lsblk -e 7,11`)
- The live ISO boot medium (identified by `/run/archiso/bootmnt` in `/proc/mounts`, parent disk via `lsblk PKNAME`)

`CheckInternet()` retries up to 3 times with a 3-second timeout per attempt (2-second pause between) before failing. A single dropped packet does not abort the installation.

`DetectWindows()` runs as a Soft step after `create-users`:
1. Runs `os-prober` on the live system; parses output for lines where the label or shortname contains "windows"
2. Falls back to checking `os.Stat("/mnt/boot/efi/EFI/Microsoft/Boot/bootmgfw.efi")` (UEFI systems only)
3. Sets `cfg.WindowsEFIPath` to a non-empty value if Windows is found; bootloader step uses it

---

## Style System

`style/styles.go` uses the **Catppuccin Mocha** palette.

```go
style.StyleStepHeader     // bold section headers
style.StyleKey            // field labels
style.StyleValue          // field values
style.StyleSelected       // highlighted / active element
style.StyleMuted          // secondary / dimmed text
style.StyleGood           // green (dry-run prefix, success)
style.StyleWarn           // yellow warning
style.StyleError          // red error
style.StyleDanger         // red danger (disk wipe warning)
style.StyleButtonActive / StyleButtonInactive / StyleButtonDanger

style.Checkbox(checked bool) string
style.HelpRow("key", "action", ...) string
style.ProgressBar(width int, pct float64) string
```

---

## Dry-Run Mode

`./falconia --dry-run` — no root required. Every `RunDry` / `RunChrootDry` call logs `[DRY RUN] Would execute: <cmd>` with a 100 ms simulated delay. Explicit `cfg.DryRun` guards cover file writes, blkid calls, and anything that doesn't go through those wrappers.

The full TUI Phase 1 and Phase 2 pipeline are testable without a real disk.
