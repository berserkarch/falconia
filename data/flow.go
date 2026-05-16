// Package data -- handles the flow
package data

import "falconia/config"

// StepKey is a typed identifier for each install step.
// Adding a new step: declare a constant here, add a StepDef to Pipeline,
// then wire the implementation in tui/steps/progress.go.
type StepKey string

const (
	StepVerifyInternet StepKey = "verify-internet"
	StepSyncClock      StepKey = "sync-clock"
	StepRankMirrors    StepKey = "rank-mirrors"
	StepPartitionDisk  StepKey = "partition-disk"
	StepFormatDisks    StepKey = "format-disks"
	StepMountDisks     StepKey = "mount-disks"
	StepCopyLiveFiles  StepKey = "copy-live-files"
	StepPacstrap       StepKey = "pacstrap"
	StepGenFstab       StepKey = "gen-fstab"
	StepSetTimezone    StepKey = "set-timezone"
	StepWriteCrypttab  StepKey = "write-crypttab"
	StepSetLocale      StepKey = "set-locale"
	StepConfigNetwork  StepKey = "config-network"
	StepSetHostname    StepKey = "set-hostname"
	StepRootPassword   StepKey = "root-password"
	StepCreateUsers    StepKey = "create-users"
	StepBootloader     StepKey = "bootloader"
	StepInstallDesktop StepKey = "install-desktop"
	StepInstallPkgs    StepKey = "install-packages"
	StepInstallDrivers StepKey = "install-drivers"
	StepDetectWindows  StepKey = "detect-windows"
	StepEnableServices StepKey = "enable-services"
	StepPostCleanup    StepKey = "post-cleanup"
	StepCleanup        StepKey = "cleanup"
)

// StepDef is one entry in the install pipeline.
// When is nil means the step always runs.
// Soft steps log a warning on failure and continue instead of aborting.
type StepDef struct {
	Key   StepKey
	Label string
	When  func(*config.InstallConfig) bool
	Soft  bool
}

// Pipeline is the ordered install sequence — the equivalent of Calamares
// settings.conf. Edit here to add, remove, or reorder steps.
// Implementations are registered in tui/steps/progress.go.
var Pipeline = []StepDef{
	{Key: StepVerifyInternet, Label: "Verify internet"},
	{Key: StepSyncClock, Label: "Sync system clock", Soft: true},
	{Key: StepRankMirrors, Label: "Rank mirrors", Soft: true, When: func(c *config.InstallConfig) bool {
		return c.RankMirrors
	}},
	{Key: StepPartitionDisk, Label: "Partition disk"},
	{Key: StepFormatDisks, Label: "Format filesystems"},
	{Key: StepMountDisks, Label: "Mount filesystems"},
	{Key: StepCopyLiveFiles, Label: "Copy live environment files"},
	{Key: StepPacstrap, Label: "Install base system (pacstrap)"},
	{Key: StepGenFstab, Label: "Generate fstab"},
	{Key: StepSetTimezone, Label: "Set timezone"},
	{Key: StepWriteCrypttab, Label: "Write crypttab", When: func(c *config.InstallConfig) bool {
		return c.EncryptDisk
	}},
	{Key: StepSetLocale, Label: "Set locale"},
	{Key: StepConfigNetwork, Label: "Configure network"},
	{Key: StepSetHostname, Label: "Set hostname"},
	{Key: StepRootPassword, Label: "Set root password"},
	{Key: StepDetectWindows, Label: "Detect other OS", Soft: true},
	{Key: StepBootloader, Label: "Install bootloader"},
	{Key: StepInstallDesktop, Label: "Install desktop environment", When: func(c *config.InstallConfig) bool {
		return c.DesktopEnv != "none" && c.DesktopEnv != ""
	}},
	{Key: StepInstallPkgs, Label: "Install extra packages"},
	{Key: StepInstallDrivers, Label: "Install hardware drivers", When: func(c *config.InstallConfig) bool {
		for _, gpu := range c.Hardware.GPUs {
			if gpu.Vendor == "nvidia" {
				return true
			}
		}
		return c.Hardware.WiFi == "broadcom" ||
			(c.Hardware.VM != "none" && c.Hardware.VM != "")
	}},
	{Key: StepCreateUsers, Label: "Create users"},
	{Key: StepEnableServices, Label: "Enable services"},
	{Key: StepPostCleanup, Label: "Post-install cleanup"},
	{Key: StepCleanup, Label: "Unmount & cleanup", Soft: true},
}
