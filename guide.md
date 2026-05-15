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
├── main.go                      # Entry point
├── config/
│   └── config.go                # InstallConfig struct + helper methods
├── style/
│   └── styles.go                # Catppuccin Mocha palette, lipgloss styles, helpers
├── tui/
│   ├── app.go                   # Root bubbletea model, step routing, nav
│   ├── sidebar.go               # Live config summary panel (right side)
│   └── steps/
│       ├── messages.go          # Navigation message types (Done, Back, StartInstall)
│       ├── welcome.go           # Step 00 — logo + firmware detection
│       ├── disk.go              # Step 01 — disk, scheme, filesystem, encryption, swap
│       ├── partitions.go        # Step 01b — manual partition mapping (root/efi/swap)
│       ├── network.go           # Step 02 — ethernet / WiFi / skip
│       ├── locale.go            # Step 03 — timezone, locale, keymap
│       ├── users.go             # Steps 04-05 — hostname, root password, user creation
│       ├── selections.go        # Generic radio list; used for kernel, desktop, bootloader
│       ├── packages.go          # Step 08 — categorised extra package selector
│       ├── postinstall.go       # Step 10 — service toggles
│       ├── confirm.go           # Step 11 — full summary + install button
│       └── progress.go          # Phase 2 — install pipeline runner + log viewport
└── installer/
    ├── runner.go                # Run, RunDry, RunChroot, RunChrootDry, RunSh, runWithStdin
    ├── detect.go                # DetectFirmware, ListDisks, liveIsoDisk, CheckInternet
    ├── disk.go                  # PartitionDisk, FormatDisks, MountDisks, Cleanup
    ├── base.go                  # Pacstrap (dracut, LUKS keyfile, Plymouth), GenFstab, GenCrypttab
    ├── system.go                # SetTimezone, SetLocale, ConfigureNetwork, SetHostname, CreateUsers
    ├── bootloader.go            # InstallBootloader → installGrub / installSystemdBoot
    ├── postinstall.go           # InstallDesktop, InstallPackages, EnableServices
    ├── livefiles.go             # CopyLiveFiles — copies select files from live ISO to /mnt
    ├── helpers.go               # writeChroot, appendFile, runOutput, runWithStdin
    ├── timezone.go              # ListTimezones
    └── locales.go               # ListLocales
```

---

## config.InstallConfig

Single struct passed to every installer function. Set during Phase 1, consumed during Phase 2.

| Field | Type | Description |
|---|---|---|
| `Firmware` | string | `"uefi"` or `"bios"` — auto-detected on startup |
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
| `Users` | []User | Non-root users (username, password, shell, groups) |
| `Kernel` | string | `"linux"`, `"linux-lts"`, `"linux-zen"`, `"linux-hardened"` |
| `DesktopEnv` | string | `"none"`, `"gnome"`, `"kde"`, `"xfce"`, `"hyprland"`, `"i3"` |
| `DisplayManager` | string | Auto-set from DE choice |
| `ExtraPackages` | []string | User-selected packages from the packages step |
| `Bootloader` | string | `"grub"` or `"systemd-boot"` |
| `GrubTimeout` | int | Bootloader menu timeout in seconds |
| `EnableBluetooth` | bool | Install and enable bluez |
| `EnableCups` | bool | Install and enable CUPS |
| `EnableSSH` | bool | Install and enable sshd |
| `RankMirrors` | bool | Run reflector before pacstrap |
| `ExtraServices` | map[string]bool | Dynamic service flags (docker, tailscale, flatpak, zram, etc.) |
| `MountPoints` | map[string]string | Manual partitioning: mount point → device path |
| `DryRun` | bool | Skip actual command execution |

`config.Defaults()` returns sensible starting values (guided, ext4, 4096 MiB swap, GRUB, 5s timeout, US locale).

---

## Phase 1 — TUI Configuration Steps

Managed by `tui/app.go`. Steps are visited in order; the user can go back with `esc`. The sidebar (`tui/sidebar.go`) shows a live summary of the current config at all times.

| # | File | What the user configures |
|---|---|---|
| 00 | `welcome.go` | Intro screen, detected firmware shown |
| 01 | `disk.go` | Target disk, guided/manual scheme, filesystem, encryption + passphrase, swap size (advanced) |
| 01b | `partitions.go` | Manual only — maps `/`, `/boot/efi`, `swap` to specific partitions |
| 02 | `network.go` | Ethernet / WiFi (SSID + password) / skip |
| 03 | `locale.go` | Timezone (searchable list), locale, keymap |
| 04 | `users.go` | Hostname |
| 05 | `users.go` | Root password, non-root user (name, password, shell, wheel group) |
| 06 | `selections.go` | Kernel variant |
| 07 | `selections.go` | Desktop environment |
| 08 | `packages.go` | Extra packages by category with collapsible headers |
| 09 | `selections.go` | Bootloader |
| 10 | `postinstall.go` | Service toggles — mirrors, SSH, Bluetooth, CUPS, Docker, Tailscale, Avahi, TLP, UFW, Flatpak, Snap, Zram, Trim, Microcode |
| 11 | `confirm.go` | Read-only summary; **Install** button triggers Phase 2 |

**Advanced mode** (`ctrl+a`) unlocks extra fields: swap size on the disk step, package descriptions on the packages step.

**Key nav:** `↑↓` / `tab` move between fields; `←→` change values; `enter` advances; `esc` goes back.

---

## Phase 2 — Install Pipeline

Defined in `tui/steps/progress.go` → `buildSteps()`. Steps run sequentially in a goroutine; each streams its output to a scrollable log viewport.

```
buildSteps(cfg) returns []installStep{label, func}
```

Current step order:

| # | Label | Function | Conditional |
|---|---|---|---|
| 1 | Verify internet | `CheckInternet` | always |
| 2 | Sync system clock | `SyncClock` | always |
| 3 | Rank mirrors | `RankMirrors` | `cfg.RankMirrors` |
| 4 | Partition disk | `PartitionDisk` | always |
| 5 | Format filesystems | `FormatDisks` | always |
| 6 | Mount filesystems | `MountDisks` | always |
| 7 | Copy live environment files | `CopyLiveFiles` | always |
| 8 | Install base system | `Pacstrap` | always |
| 9 | Generate fstab | `GenFstab` | always |
| 10 | Set timezone | `SetTimezone` | always |
| 11 | Write crypttab | `GenCrypttab` | `cfg.EncryptDisk` |
| 12 | Set locale | `SetLocale` | always |
| 13 | Configure network | `ConfigureNetwork` | always |
| 14 | Set hostname | `SetHostname` | always |
| 15 | Set root password | `SetRootPassword` | always |
| 16 | Create users | `CreateUsers` | always |
| 17 | Install bootloader | `InstallBootloader` | always |
| 18 | Install desktop environment | `InstallDesktop` | `cfg.DesktopEnv != "none"` |
| 19 | Install extra packages | `InstallPackages` | always |
| 20 | Enable services | `EnableServices` | always |
| 21 | Unmount & cleanup | `Cleanup` | always |

### Adding a new step

```go
// Always runs
steps = append(steps, installStep{"My step", func(c *config.InstallConfig, log installer.LineHandler) error {
    return installer.MyFunction(c, log)
}})

// Conditional
if cfg.SomeFlag {
    steps = append(steps, installStep{"My step", func(c *config.InstallConfig, log installer.LineHandler) error {
        return installer.MyFunction(c, log)
    }})
}
```

---

## installer/ Package

### runner.go — Command execution

All installer functions receive a `LineHandler func(string)` for streaming output.

```go
Run(log, "pacman", "-S", "vim")              // run on host
RunDry(cfg, log, "pacman", "-S", "vim")      // skips if cfg.DryRun
RunChroot(log, "pacman", "-S", "vim")        // run inside arch-chroot /mnt
RunChrootDry(cfg, log, "pacman", "-S", "vim")
RunSh(log, "echo hello")                     // /bin/sh -c
runWithStdin(log, reader, "cryptsetup", ...) // piped stdin (passwords)
runOutput("blkid", "-s", "UUID", ...)        // capture stdout as string
```

`RunDry` logs `[DRY RUN] Would execute: <cmd>` and returns nil without running anything.

### disk.go — Partitioning

**Partition layout (guided):**

UEFI: `sda1` EFI 512 MiB (FAT32) · `sda2` swap (optional) · `sda3` root  
BIOS: `sda1` BIOS-boot 1 MiB · `sda2` swap (optional) · `sda3` root

NVMe/mmcblk devices get a `p` suffix: `nvme0n1p1`, `nvme0n1p2`, etc.

After `parted` creates partitions, `partprobe <disk> && udevadm settle` is called to ensure the kernel sees the new layout before `mkfs` runs.

**Encryption (LUKS):** Root partition is formatted with `cryptsetup luksFormat --type luks1`, opened as `/dev/mapper/cryptroot`, then `mkfs` runs on the mapper device.

### base.go — Pacstrap + initramfs

`Pacstrap` does several things beyond just running pacstrap:

1. Runs `pacstrap /mnt <packages>`
2. Writes `/mnt/etc/dracut.conf.d/plymouth.conf` (Plymouth in initramfs)
3. If encrypted + GRUB: writes `/mnt/etc/dracut.conf.d/encryption.conf`, generates 512-byte random keyfile at `/mnt/crypto_keyfile.bin` (mode `0000`), registers it as a second LUKS key slot via `luksAddKey`
4. Runs `plymouth-set-default-theme berserk` in chroot
5. Builds initramfs with dracut (`--force`)
6. Builds fallback initramfs (`--no-hostonly --force`)

**LUKS keyfile rationale:** GRUB must unlock LUKS itself to read the kernel and initramfs (one passphrase prompt). Without a keyfile, the initramfs would prompt a second time. The keyfile is embedded in the initramfs by dracut and referenced by `crypttab`/`rd.luks.key`, so the initramfs unlocks LUKS silently. For systemd-boot, the ESP is unencrypted so the keyfile would be exposed — it is not generated and the initramfs prompts once instead.

### bootloader.go — GRUB & systemd-boot

**GRUB kernel cmdline (encrypted):**
```
quiet splash rd.luks.uuid=<uuid> rd.luks.name=<uuid>=cryptroot rd.luks.key=/crypto_keyfile.bin root=/dev/mapper/cryptroot
```

**GRUB kernel cmdline (plain):**
```
quiet splash
```

**systemd-boot** copies kernel + initramfs (+ microcode if enabled) to the ESP, writes `/boot/efi/loader/entries/arch.conf` and `/mnt/etc/kernel/cmdline` (so future kernel upgrades via `kernel-install` preserve the correct parameters).

UEFI installs also run `grub-install --removable` as a fallback for firmware that ignores NVRAM boot entries.

### livefiles.go — Copying from the live ISO

Files listed in `liveFiles` are copied from the running live environment into `/mnt` before pacstrap, preserving directory structure and file permissions.

```go
var liveFiles = []string{
    "/etc/pacman.conf",
    // add more paths here
}
```

`CopyLiveFiles` runs at step 7, before pacstrap, so the custom `pacman.conf` (with BerserkArch repos) is in place when packages are installed.

### detect.go — Hardware detection

`ListDisks()` excludes:
- Loop devices and optical drives (`lsblk -e 7,11`)
- The live ISO boot medium (identified by `/run/archiso/bootmnt` in `/proc/mounts`, resolved to the parent disk via `lsblk PKNAME`)

---

## Style System

`style/styles.go` uses the **Catppuccin Mocha** palette.

```go
// Pre-defined styles
style.StyleStepHeader   // bold section headers in steps
style.StyleKey          // field labels
style.StyleValue        // field values
style.StyleSelected     // highlighted / active element
style.StyleMuted        // secondary / dimmed text
style.StyleError        // red error messages
style.StyleDanger       // red warning (e.g. disk wipe warning)
style.StyleGood         // green success/dry-run prefix
style.StyleButtonActive / StyleButtonInactive / StyleButtonDanger

// Helpers
style.Checkbox(checked bool) string          // [x] or [ ]
style.HelpRow("key", "action", ...)  string  // bottom help bar
style.ProgressBar(pct float64, width int) string
```

---

## Dry-Run Mode

Run with `./falconia --dry-run` (no root required). Every `RunDry` / `RunChrootDry` call logs `[DRY RUN] Would execute: <cmd>` with a 100 ms simulated delay instead of running the command. Explicit `cfg.DryRun` guards in installer functions handle anything that doesn't go through those wrappers (file writes, blkid calls, etc.).

This makes the full TUI walkable and the entire Phase 2 pipeline testable without a real disk.
