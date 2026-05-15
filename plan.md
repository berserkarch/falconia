# Falconia — Implementation Plan

## Vision

Falconia is a TUI installer for BerserkArch — an Arch Linux-based distro. It does not try to be a generic installer. It is Arch-specific, BerserkArch-opinionated, and ships as a single binary with zero external config files.

The reference for what a mature Arch installer does is the BerserkArch Calamares fork in `cal/`. Falconia takes the same installation logic but replaces the GUI with a TUI, replaces shell scripts and `.conf` modules with Go code, and replaces every external data file (package lists, service lists, driver mappings) with Go source files that compile into the binary.

---

## Core Philosophy

**1. Single binary, zero config files**
All data — package lists, DE packages, driver mappings, service lists, kernel params — lives in `.go` files inside a `data/` package. No `.conf`, no `.yaml`, no external scripts. Everything compiles in.

**2. Arch-specific, not generic**
We rely directly on `parted`, `mkfs.*`, `pacstrap`, `arch-chroot`, `genfstab`, `dracut`, `cryptsetup`, `bootctl`, `grub-install`. No abstraction layers, no compatibility shims for other distros.

**3. Automatic where possible, explicit where it matters**
Hardware is detected silently at startup. The user never picks "install NVIDIA drivers" — the installer detects the GPU and handles it. The user does make choices that are genuinely theirs: disk, encryption, DE, kernel, bootloader.

**4. Dry-run always works**
Every installer function respects `cfg.DryRun`. The full Phase 2 pipeline must be walkable without root.

---

## Current State (what's already built)

- TUI skeleton: bubbletea + lipgloss, step routing, sidebar, advanced mode
- Disk step: disk picker (live ISO excluded), guided/manual scheme, ext4/btrfs/xfs, encryption passphrase, swap size (partition only)
- Guided partitioning: UEFI (EFI + optional swap + root) and BIOS (BIOS-boot + optional swap + root), NVMe/mmcblk suffix handling, `partprobe` + `udevadm settle` after parted
- LUKS: `luksFormat`, `luksOpen`, keyfile generation + `luksAddKey`, dracut `crypt` module, `crypttab`, keyfile skipped for systemd-boot
- Filesystem format + mount
- Pacstrap with base + BerserkArch packages
- Plymouth: dracut module, `plymouth-set-default-theme berserk`, `quiet splash` in both bootloaders
- Locale, timezone, keymap, hostname, users
- GRUB (UEFI + BIOS, removable fallback) and systemd-boot
- Extra packages (categorised TUI picker)
- Services (NetworkManager always + optional flags)
- Copy live files (`/etc/pacman.conf`)
- Conditional pipeline steps (`RankMirrors`, `EncryptDisk`, `DesktopEnv`)
- Dry-run mode throughout

---

## What We Are Building

Gaps identified from the Calamares reference and the answers above, in priority order:

1. **Hardware detection subsystem** — CPU, GPU (NVIDIA/AMD/Intel), WiFi (Broadcom), VM type
2. **Driver installation** — NVIDIA (open vs proprietary), Broadcom WiFi, VM guest tools
3. **Swap overhaul** — modes: none, partition, file, suspend-sized (= RAM)
4. **Windows / dual-boot detection** — os-prober, GRUB/systemd-boot entry
5. **Post-install cleanup** — .pacnew, unnecessary microcode, machine ID
6. **Data package** — all static data (packages, services, driver maps) as Go files
7. **Btrfs subvolumes** — `/@`, `/@home`, `/@cache`, `/@log` (last, after everything else)

---

## Architecture

### Package layout (target state)

```
falconia/
├── main.go
├── config/
│   └── config.go           # InstallConfig, User, HardwareProfile structs
├── data/                   # NEW — all static data as Go files
│   ├── packages.go         # BasePackages, DEPackages, DriverPackages, VMPackages
│   ├── services.go         # EnabledServices per scenario
│   └── kernelparams.go     # Kernel cmdline params per scenario
├── style/
│   └── styles.go
├── tui/
│   ├── app.go
│   ├── sidebar.go
│   └── steps/
│       ├── welcome.go      # Shows detected hardware
│       ├── disk.go         # + swap mode picker
│       ├── partitions.go
│       ├── network.go
│       ├── locale.go
│       ├── users.go
│       ├── selections.go
│       ├── packages.go
│       ├── postinstall.go
│       ├── confirm.go      # Shows hardware-driven changes
│       ├── messages.go
│       └── progress.go
└── installer/
    ├── runner.go
    ├── detect.go           # + hardware detection
    ├── disk.go             # + swap modes, btrfs subvols (later)
    ├── base.go
    ├── system.go
    ├── bootloader.go       # + Windows boot entry, kernel params
    ├── postinstall.go      # + drivers, VM tools, cleanup
    ├── livefiles.go
    ├── helpers.go
    ├── locales.go
    └── timezone.go
```

### config.InstallConfig additions

```go
// Hardware (auto-detected at startup, never user-set)
Hardware HardwareProfile

// Disk
SwapMode    SwapMode // "none" | "partition" | "file" | "suspend"
SwapSize    int      // MiB; used for "partition" and "file" modes

// Dual-boot
DualBootWindows bool // detected, shown to user for confirmation

// Machine ID (generated during install)
MachineID string
```

### HardwareProfile (new, in config/)

```go
type HardwareProfile struct {
    CPU        string    // "intel" | "amd" | "other"
    GPUs       []GPUInfo
    WiFi       string    // "broadcom" | "other"
    VM         string    // "virtualbox" | "vmware" | "qemu" | "kvm" | "none"
    RAMBytes   int64     // for suspend-sized swap calculation
    HasNVMe    bool
}

type GPUInfo struct {
    Vendor  string // "nvidia" | "amd" | "intel"
    UseOpen bool   // true = nvidia-open; false = nvidia (proprietary)
}
```

`HardwareProfile` is populated once at startup by `installer.DetectHardware()`, stored in `cfg.Hardware`, and never changes after that.

### SwapMode (new, in config/)

```go
type SwapMode string

const (
    SwapNone      SwapMode = "none"       // no swap at all
    SwapPartition SwapMode = "partition"  // dedicated partition (current behaviour)
    SwapFile      SwapMode = "file"       // /swapfile inside root partition
    SwapSuspend   SwapMode = "suspend"    // partition or file sized to RAM (for hibernate)
)
```

### data/ package

Every list that would be a `.conf` file in Calamares becomes a Go file here. Examples:

```go
// data/packages.go
var Base = []string{"base", "base-devel", "linux-firmware", ...}

var ByDE = map[string][]string{
    "kde":      {"plasma", "sddm", ...},
    "gnome":    {"gnome", "gdm", ...},
    "xfce":     {"xfce4", "lightdm", ...},
    "hyprland": {"hyprland", "sddm", ...},
    "i3":       {"i3-wm", "lightdm", ...},
}

var ByDriver = map[string][]string{
    "nvidia":          {"nvidia", "nvidia-utils", "nvidia-settings"},
    "nvidia-open":     {"nvidia-open", "nvidia-utils", "nvidia-settings"},
    "broadcom":        {"broadcom-wl"},
    "broadcom-dkms":   {"broadcom-wl-dkms"},
}

var ByVM = map[string][]string{
    "virtualbox": {"virtualbox-guest-utils"},
    "vmware":     {"open-vm-tools"},
    "qemu":       {"qemu-guest-agent", "spice-vdagent"},
}
```

```go
// data/services.go
var AlwaysEnable = []string{"NetworkManager", "fstrim.timer"}

var ByVM = map[string][]string{
    "virtualbox": {"vboxservice"},
    "vmware":     {"vmtoolsd", "vmware-vmblock-fuse"},
    "qemu":       {"qemu-guest-agent"},
}

var ByDE = map[string][]string{
    "kde":      {"sddm"},
    "gnome":    {"gdm"},
    "xfce":     {"lightdm"},
    "hyprland": {"sddm"},
    "i3":       {"lightdm"},
}
```

```go
// data/kernelparams.go
// Appended to every install regardless of hardware
var Always = []string{"quiet", "splash", "nowatchdog"}

// Appended when NVMe detected
var NVMe = []string{"nvme_load=YES"}
```

---

## TUI — Phase 1 Steps

Steps are in order. User navigates forward with `enter`, backward with `esc`. `ctrl+a` toggles advanced mode. The sidebar shows a live config summary throughout.

### Step 00 — Welcome
`tui/steps/welcome.go`

**Currently:** Logo + detected firmware.
**After:** Also shows the hardware detection results so the user knows what was found before they start.

Display:
- Detected firmware (UEFI/BIOS)
- CPU: Intel / AMD
- GPU(s): vendor + driver that will be installed
- WiFi: if Broadcom detected, note that `broadcom-wl` will be installed
- VM: if running inside a VM, note guest tools that will be installed
- RAM: shown for reference (informs swap suggestions)

Hardware detection runs before the TUI starts (in `main.go`, before `tea.NewProgram`). Results go into `cfg.Hardware`.

### Step 01 — Disk
`tui/steps/disk.go`

**Currently:** Disk picker, guided/manual, filesystem, encrypt on/off + passphrase, swap size (MiB).
**After:** Replace the swap size field with a swap mode picker:

```
Swap    [ none ]  [ partition ]  [ file ]  [ suspend ]
Size    4096 MiB   (shown only for partition / file / suspend)
```

Suspend mode auto-fills size = detected RAM (shown as read-only label). Partition and file modes show the size input. None hides it.

Encryption is disabled (greyed out) when scheme is manual, same as now.

### Step 01b — Manual Partitions
`tui/steps/partitions.go`

No change planned. Maps `/`, `/boot/efi`, `swap` to partition devices.

### Step 02 — Network
`tui/steps/network.go`

No change planned.

### Step 03 — Locale
`tui/steps/locale.go`

**Future nice-to-have:** GeoIP hint shown next to the timezone picker ("Detected: Asia/Kolkata"). User can accept or change. Not blocking.

### Steps 04-05 — Hostname & Users
`tui/steps/users.go`

Default shell is already `/bin/zsh`. Default groups should include `docker` and `wireshark` in addition to `wheel`, matching the Calamares config. Change in `Save()`.

### Step 06 — Kernel
`tui/steps/selections.go`

Note: when Broadcom WiFi is detected and the user picks `linux-lts`, the installer must use `broadcom-wl-dkms` instead of `broadcom-wl` (non-DKMS package only tracks the vanilla `linux` kernel). This is handled automatically in the install pipeline — the TUI step itself doesn't change.

### Step 07 — Desktop
`tui/steps/selections.go`

Add **Cinnamon** as an option (it's in the Calamares config but missing from Falconia).

### Step 08 — Packages
`tui/steps/packages.go`

No structural change. Package list can be expanded.

### Step 09 — Bootloader
`tui/steps/selections.go`

No change planned (GRUB + systemd-boot).

### Step 10 — Post-install Services
`tui/steps/postinstall.go`

Hardware-detected services (NVIDIA, Broadcom, VM tools) are **not shown here** — they are automatic. This step remains for user-optional things (SSH, Bluetooth, CUPS, Docker, Tailscale, etc.).

Remove the `microcode` toggle — microcode is now always handled automatically based on `cfg.Hardware.CPU`. No user choice needed.

### Step 11 — Confirm
`tui/steps/confirm.go`

Add a "Hardware" section showing what the installer will do automatically:
- "NVIDIA GPU detected → will install `nvidia-open`"
- "Running in VirtualBox → will install `virtualbox-guest-utils`"
- "Broadcom WiFi detected → will install `broadcom-wl`"
- "Windows installation detected on /dev/sdaX → will add boot entry"

This gives the user full visibility before they hit Install.

---

## Install Pipeline — Phase 2

`tui/steps/progress.go` → `buildSteps()`

Steps in order. Conditional steps are marked with their condition.

```
 1  Verify internet
 2  Sync system clock
 3  Rank mirrors                    [cfg.RankMirrors]
 4  Partition disk
 5  Format filesystems
 6  Mount filesystems
 7  Copy live environment files
 8  Install base system (pacstrap)
 9  Generate machine ID
10  Generate fstab
11  Write crypttab                  [cfg.EncryptDisk]
12  Set timezone
13  Set locale
14  Configure network               (no-op if ethernet/skip)
15  Set hostname
16  Set root password
17  Create users
18  Install bootloader
19  Install desktop environment     [cfg.DesktopEnv != "none"]
20  Install extra packages
21  Install NVIDIA drivers          [hardware.GPU == nvidia]
22  Install Broadcom WiFi driver    [hardware.WiFi == broadcom]
23  Install VM guest tools          [hardware.VM != none]
24  Add Windows boot entry          [cfg.DualBootWindows]
25  Enable services
26  Post-install cleanup
27  Unmount & cleanup
```

---

## Subsystem: Hardware Detection

`installer/detect.go` — new function `DetectHardware() HardwareProfile`

Called once in `main.go` before the TUI starts. Populates `cfg.Hardware`.

### CPU Detection
Read `/proc/cpuinfo`:
- `GenuineIntel` → `"intel"`, will install `intel-ucode`
- `AuthenticAMD` → `"amd"`, will install `amd-ucode`

Microcode is now always installed based on this detection. Remove the `ExtraServices["microcode"]` toggle.

### GPU Detection
Parse `lspci -nn` output for VGA/3D/Display class devices:
- `10de:` prefix → NVIDIA
  - Check if `nvidia-open` supports the specific device ID → set `UseOpen`
  - If not supported → use `nvidia` (proprietary)
- `1002:` prefix → AMD (radeon/amdgpu, open source, no extra package needed)
- `8086:` prefix → Intel (i915/iris, open source, no extra package needed)

Multiple GPUs are supported (e.g., hybrid Intel + NVIDIA laptop).

### WiFi Detection
Parse `lspci -nn` for Network class:
- Broadcom vendor ID `14e4:` → `"broadcom"`
  - LTS kernel → `broadcom-wl-dkms`
  - Other → `broadcom-wl`

### VM Detection
Read `/sys/class/dmi/id/product_name` and `/sys/class/dmi/id/sys_vendor`:
- `VirtualBox` → `"virtualbox"`
- `VMware` → `"vmware"`
- Check `/sys/hypervisor/type` or `systemd-detect-virt` for QEMU/KVM

### RAM Size
Read `/proc/meminfo` for `MemTotal`. Used to size suspend swap.

### NVMe Detection
Check if any disk in `/dev/nvme*` exists → `HasNVMe = true`. Adds `nvme_load=YES` to kernel cmdline.

### Windows Detection
Separate function `DetectWindows() (bool, string)` called during the install pipeline (not at startup, since the target disk may not be selected yet). Runs after partitioning. Uses `os-prober` or directly reads partition types from the disk layout. Returns whether Windows was found and on which partition.

---

## Subsystem: Swap Overhaul

`installer/disk.go`

### Partition mode (current behaviour, refactored)
A dedicated swap partition is created between EFI/BIOS-boot and root. Size = `cfg.SwapSize` MiB. Behaviour unchanged.

### File mode
No swap partition is created. After the root filesystem is mounted and pacstrap completes:
1. `fallocate -l <size>M /mnt/swapfile`
2. `chmod 600 /mnt/swapfile`
3. `mkswap /mnt/swapfile`
4. `swapon /mnt/swapfile`
5. `genfstab` will pick it up automatically via `swapon`

Btrfs note: swap files on btrfs require `chattr +C /mnt/swapfile` (no CoW) before `mkswap`. Handle this in `FormatDisks` when filesystem is btrfs and swap mode is file.

### Suspend mode
Same as file mode but `cfg.SwapSize` is set automatically to `cfg.Hardware.RAMBytes / 1024 / 1024` (MiB) in the disk step's `Save()`. The kernel cmdline must also include `resume=/dev/mapper/cryptroot` (encrypted) or `resume=UUID=<uuid>` (plain) and `resume_offset=<offset>` for swap files. The offset is obtained via `filefrag -v /mnt/swapfile` after creation.

### None mode
No swap partition, no swap file. Partition layout skips the swap step.

### Partition layout by swap mode

**UEFI:**
```
Swap mode     sda1          sda2          sda3
partition     EFI 512MiB    swap NMiB     root (rest)
file/suspend  EFI 512MiB    root (rest)   —
none          EFI 512MiB    root (rest)   —
```

**BIOS:**
```
Swap mode     sda1          sda2          sda3
partition     BIOS 1MiB     swap NMiB     root (rest)
file/suspend  BIOS 1MiB     root (rest)   —
none          BIOS 1MiB     root (rest)   —
```

The `rootPartition()` helper in `bootloader.go` already handles the `SwapSize > 0` condition for partition number. Update it to check `SwapMode == SwapPartition` instead.

---

## Subsystem: Driver Installation

### NVIDIA
`installer/postinstall.go` — `InstallNvidiaDriver(cfg, log)`

```
if UseOpen:
    pacman -S nvidia-open nvidia-utils nvidia-settings
else:
    pacman -S nvidia nvidia-utils nvidia-settings
```

For hybrid (Intel + NVIDIA) laptops: also install `nvidia-prime`.

After install:
- Add `nvidia-drm.modeset=1` to kernel cmdline (requires regenerating grub.cfg or updating /etc/kernel/cmdline for systemd-boot)
- Enable `nvidia-persistenced` service
- Generate a new initramfs (dracut) to include the nvidia module

### Broadcom WiFi
`installer/postinstall.go` — `InstallBroadcomDriver(cfg, log)`

```
if kernel == "linux-lts":
    pacman -S broadcom-wl-dkms
else:
    pacman -S broadcom-wl
```

After install: `modprobe wl` (or reboot handles it).

### VM Guest Tools
`installer/postinstall.go` — `InstallVMTools(cfg, log)`

```
virtualbox:
    pacman -S virtualbox-guest-utils
    systemctl enable vboxservice

vmware:
    pacman -S open-vm-tools
    systemctl enable vmtoolsd vmware-vmblock-fuse

qemu/kvm:
    pacman -S qemu-guest-agent spice-vdagent
    systemctl enable qemu-guest-agent
```

VM installs also: remove `power-profiles-daemon` (power management is irrelevant in a VM).

---

## Subsystem: Windows / Dual-Boot

`installer/bootloader.go` — `AddWindowsBootEntry(cfg, log)`

Detection runs as an install step after partitioning and mounting. Uses `os-prober` (install it if not present on live ISO, or ship it in base packages).

`os-prober` writes to stdout lines like:
```
/dev/sda1:Windows 10:Windows:chain
```

Parse this. If a Windows entry is found, set `cfg.DualBootWindows = true` and store the partition.

**GRUB:** `grub-mkconfig` already calls os-prober if it's installed and `GRUB_DISABLE_OS_PROBER=false` is set. Ensure this is set in `/etc/default/grub`. No extra step needed — grub-mkconfig handles it.

**systemd-boot:** Write a manual loader entry:
```
# /boot/efi/loader/entries/windows.conf
title   Windows
efi     /EFI/Microsoft/Boot/bootmgfw.efi
```

---

## Subsystem: Post-Install Cleanup

`installer/postinstall.go` — `Cleanup(cfg, log)` (currently just unmount/swapoff)

Rename the current `Cleanup` to `Unmount`. Create a new `PostInstallCleanup` step that runs before unmount:

### Machine ID
`systemd-machine-id-setup` inside chroot. Generates `/etc/machine-id`. Required for proper systemd and D-Bus operation.

### Unnecessary microcode removal
If CPU is Intel → remove `amd-ucode` if installed (pacstrap may have pulled it).
If CPU is AMD → remove `intel-ucode`.
Use `pacman -R --noconfirm` inside chroot.

### .pacnew cleanup
Find and remove `.pacnew` files left by pacman:
```bash
find /mnt -name "*.pacnew" -delete
```

### Log journal persistence
Change journal from volatile (live ISO default) to auto (persistent on disk):
```
sed -i 's/#Storage=auto/Storage=auto/' /mnt/etc/systemd/journald.conf
```

### fstrim.timer
Always enable `fstrim.timer` for SSD health. Add to the always-on services list.

---

## Subsystem: Kernel Cmdline

Kernel parameters should be assembled from the `data/kernelparams.go` data file based on hardware detection, not hardcoded in `bootloader.go`.

```go
// data/kernelparams.go

// Always added
var Always = []string{"quiet", "splash", "nowatchdog"}

// Added when NVMe detected
var NVMe = []string{"nvme_load=YES"}

// Added when NVIDIA GPU present
var Nvidia = []string{"nvidia-drm.modeset=1"}
```

`buildKernelParams(cfg) string` in `installer/bootloader.go`:
```go
func buildKernelParams(cfg *config.InstallConfig) string {
    params := append([]string{}, data.Always...)
    if cfg.Hardware.HasNVMe {
        params = append(params, data.NVMe...)
    }
    // encryption params added inline in installGrub/installSystemdBoot
    return strings.Join(params, " ")
}
```

---

## Implementation Phases

Work in this order. Each phase should leave the installer in a working, releasable state.

### Phase A — Data Package + Refactor (foundation)
1. Create `data/` package with `packages.go`, `services.go`, `kernelparams.go`
2. Move package lists out of `base.go` and `postinstall.go` into `data/`
3. Move service lists out of `postinstall.go` into `data/`
4. Replace `ExtraServices["microcode"]` with automatic CPU-based microcode (always on)
5. Remove microcode toggle from the postinstall TUI step
6. Add `docker` and `wireshark` to default user groups in `users.go`
7. Add Cinnamon to desktop options in `selections.go`

### Phase B — Swap Overhaul
1. Add `SwapMode` type to `config/config.go`
2. Update disk TUI step (`disk.go`) to show swap mode picker
3. Update `PartitionDisk` to skip swap partition for file/none/suspend modes
4. Update `FormatDisks` for file swap creation after root format
5. Update `MountDisks` for swap file activation
6. Update `rootPartition()` to use `SwapMode == SwapPartition` check
7. Handle suspend mode size + resume kernel param
8. Handle btrfs + swap file (chattr +C)

### Phase C — Hardware Detection
1. Write `DetectHardware()` in `installer/detect.go`
   - CPU detection (existing, refactor from `detectMicrocode`)
   - GPU detection via `lspci`
   - WiFi detection via `lspci`
   - VM detection via DMI + systemd-detect-virt
   - RAM size from `/proc/meminfo`
   - NVMe detection
2. Add `HardwareProfile` to `config/config.go`
3. Call `DetectHardware()` in `main.go` before TUI starts
4. Update `welcome.go` to display hardware findings
5. Update `confirm.go` to show hardware-driven install actions
6. Wire `nvme_load=YES` into kernel cmdline via `buildKernelParams`
7. Wire microcode package selection off `hardware.CPU`

### Phase D — Driver Installation
1. Write `InstallNvidiaDriver(cfg, log)` in `installer/postinstall.go`
2. Write `InstallBroadcomDriver(cfg, log)` in `installer/postinstall.go`
3. Write `InstallVMTools(cfg, log)` in `installer/postinstall.go`
4. Add conditional steps to `buildSteps()` in `progress.go`
5. Handle NVIDIA: `nvidia-drm.modeset=1` in cmdline, dracut initramfs regeneration

### Phase E — Windows / Dual-Boot
1. Add `os-prober` to base packages
2. Write `DetectWindows() (bool, string)` in `installer/detect.go`
3. Add `DualBootWindows bool` and `WindowsPartition string` to `InstallConfig`
4. Add "Detect Windows" install step in `buildSteps()`
5. Write `AddWindowsBootEntry(cfg, log)` in `installer/bootloader.go`
   - GRUB: ensure `GRUB_DISABLE_OS_PROBER=false` in `/etc/default/grub`
   - systemd-boot: write manual entry file
6. Show Windows detection result on confirm screen

### Phase F — Post-Install Cleanup
1. Rename current `Cleanup` to `Unmount` (just the umount/swapoff/cryptsetup close part)
2. Write `PostInstallCleanup(cfg, log)` covering: machine ID, microcode removal, .pacnew, journal persistence
3. Add to `buildSteps()` before `Unmount`
4. Add `fstrim.timer` to always-on services in `data/services.go`

### Phase G — Btrfs Subvolumes (last)
1. Update `FormatDisks` to detect btrfs and create named subvolumes
   - Mount btrfs, create `/@`, `/@home`, `/@cache`, `/@log`, unmount, remount at subvol=/@
2. Update `MountDisks` to mount each subvolume at its correct path
3. Handle swap file on btrfs (chattr +C, offset calculation for suspend)
4. Ensure `genfstab` correctly captures all subvolume mounts

---

## Open Questions

- **EFI mount point:** Currently `/boot/efi`. Calamares uses `/efi`. Both work for GRUB (only the EFI binary lives on the ESP). For systemd-boot the kernel lives on the ESP. No change planned for now, but worth revisiting if we ever move the kernel to the ESP for GRUB too.

- **refind:** Calamares supports it. Not planned for Falconia. GRUB + systemd-boot covers the real-world use cases.

- **Offline install:** Out of scope for now. Online (pacstrap) only.

- **GeoIP timezone detection:** Nice to have for locale step. Not blocking. Can be added to Phase A or later as a small touch.

- **NVIDIA UseOpen detection:** Requires checking device IDs against the `nvidia-open` supported hardware list. A good-enough heuristic: RTX 2000 series and newer (Turing+) support `nvidia-open`. Check the GPU device ID prefix range. Worst case, default to `nvidia` (proprietary) and let the user update post-install.

- **Surface / marvell firmware:** Calamares installs marvell firmware for Surface WiFi devices. Narrow edge case — skip for now, revisit if reported.
