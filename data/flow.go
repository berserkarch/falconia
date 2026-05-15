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
	StepEnableServices StepKey = "enable-services"
	StepCleanup        StepKey = "cleanup"
)

// StepDef is one entry in the install pipeline.
// When is nil means the step always runs.
type StepDef struct {
	Key   StepKey
	Label string
	When  func(*config.InstallConfig) bool
}

// Pipeline is the ordered install sequence — the equivalent of Calamares
// settings.conf. Edit here to add, remove, or reorder steps.
// Implementations are registered in tui/steps/progress.go.
var Pipeline = []StepDef{
	{StepVerifyInternet, "Verify internet", nil},
	{StepSyncClock, "Sync system clock", nil},
	{StepRankMirrors, "Rank mirrors", func(c *config.InstallConfig) bool {
		return c.RankMirrors
	}},
	{StepPartitionDisk, "Partition disk", nil},
	{StepFormatDisks, "Format filesystems", nil},
	{StepMountDisks, "Mount filesystems", nil},
	{StepCopyLiveFiles, "Copy live environment files", nil},
	{StepPacstrap, "Install base system (pacstrap)", nil},
	{StepGenFstab, "Generate fstab", nil},
	{StepSetTimezone, "Set timezone", nil},
	{StepWriteCrypttab, "Write crypttab", func(c *config.InstallConfig) bool {
		return c.EncryptDisk
	}},
	{StepSetLocale, "Set locale", nil},
	{StepConfigNetwork, "Configure network", nil},
	{StepSetHostname, "Set hostname", nil},
	{StepRootPassword, "Set root password", nil},
	{StepCreateUsers, "Create users", nil},
	{StepBootloader, "Install bootloader", nil},
	{StepInstallDesktop, "Install desktop environment", func(c *config.InstallConfig) bool {
		return c.DesktopEnv != "none" && c.DesktopEnv != ""
	}},
	{StepInstallPkgs, "Install extra packages", nil},
	{StepInstallDrivers, "Install hardware drivers", func(c *config.InstallConfig) bool {
		for _, gpu := range c.Hardware.GPUs {
			if gpu.Vendor == "nvidia" {
				return true
			}
		}
		return c.Hardware.WiFi == "broadcom" ||
			(c.Hardware.VM != "none" && c.Hardware.VM != "")
	}},
	{StepEnableServices, "Enable services", nil},
	{StepCleanup, "Unmount & cleanup", nil},
}
