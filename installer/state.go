package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"falconia/config"
	"falconia/data"
)

// StateVersion is bumped whenever State changes incompatibly.
const StateVersion = 1

const (
	stateDir        = "/tmp/falconia"
	tmpStatePath    = stateDir + "/state.json"
	dryRunStatePath = stateDir + "/state-dryrun.json"

	// diskStateDir is the copy on the target root filesystem. It survives a
	// reboot and is written once the filesystems are mounted.
	diskStateDir  = "/mnt/var/lib/falconia"
	diskStateFile = "var/lib/falconia/state.json" // relative to the root filesystem

	// espStubName is the file on the unencrypted ESP that lets a resume after
	// reboot find the root partition and unlock it without prompting.
	espStubName = "falconia-resume.json"
	espStubPath = "/mnt/boot/efi/" + espStubName
)

// State is everything needed to resume an interrupted installation.
// It contains every password from the config in plain text.
type State struct {
	Version   int
	UpdatedAt time.Time
	Config    *config.InstallConfig
	Completed []data.StepKey

	// PartUUIDs maps a partition role ("root", "efi", "swap", or a manual
	// mount point) to its PARTUUID, so the right partitions are found even if
	// device names change across reboots. Captured once partitioning is done.
	PartUUIDs map[string]string
}

// espStub is the minimal record kept on the ESP: where root is and, for
// encrypted installs, the passphrase to unlock it.
type espStub struct {
	Version        int
	RootPartUUID   string
	EncryptionPass string
}

// NewState starts a fresh state for cfg.
func NewState(cfg *config.InstallConfig) *State {
	return &State{Version: StateVersion, Config: cfg}
}

// IsDone reports whether step k finished in a previous run.
func (s *State) IsDone(k data.StepKey) bool {
	return slices.Contains(s.Completed, k)
}

// MarkDone records step k as finished.
func (s *State) MarkDone(k data.StepKey) {
	if !s.IsDone(k) {
		s.Completed = append(s.Completed, k)
	}
}

func localStatePath(dryRun bool) string {
	if dryRun {
		return dryRunStatePath
	}
	return tmpStatePath
}

// SaveState writes the state to /tmp and, once the target is mounted, to the
// root filesystem and the ESP.
func SaveState(s *State) error {
	s.Version = StateVersion
	s.UpdatedAt = time.Now()
	cfg := s.Config

	if s.PartUUIDs == nil && !cfg.DryRun && s.IsDone(data.StepPartitionDisk) {
		uuids, err := capturePartUUIDs(cfg)
		if err != nil {
			return err
		}
		s.PartUUIDs = uuids
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	if err := writePrivate(localStatePath(cfg.DryRun), b); err != nil {
		return err
	}

	if cfg.DryRun || !s.IsDone(data.StepMountDisks) || !isMountpoint("/mnt") {
		return nil
	}
	if err := writePrivate(filepath.Join(diskStateDir, "state.json"), b); err != nil {
		return err
	}

	if isMountpoint("/mnt/boot/efi") && s.PartUUIDs["root"] != "" {
		stub := espStub{Version: StateVersion, RootPartUUID: s.PartUUIDs["root"]}
		if cfg.EncryptDisk {
			stub.EncryptionPass = cfg.EncryptionPass
		}
		sb, err := json.Marshal(stub)
		if err != nil {
			return fmt.Errorf("encode esp stub: %w", err)
		}
		// FAT has no permissions; the mode is best effort.
		if err := writePrivate(espStubPath, sb); err != nil {
			return err
		}
	}
	return nil
}

// RemoveState deletes every copy of the state (and other resume-only files)
// once the install has finished.
func RemoveState(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.DryRun {
		log(styleGood("[DRY RUN] Would remove: ") + diskStateDir + ", " + espStubPath)
		return removeIfExists(log, dryRunStatePath)
	}
	for _, path := range []string{espStubPath, grubDefaultOrig, tmpStatePath} {
		if err := removeIfExists(log, path); err != nil {
			return err
		}
	}
	if err := os.RemoveAll(diskStateDir); err != nil {
		return fmt.Errorf("remove %s: %w", diskStateDir, err)
	}
	log("$ rm -rf " + diskStateDir)
	return nil
}

func removeIfExists(log LineHandler, path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	log("$ rm " + path)
	return nil
}

// writePrivate atomically writes b to path with mode 0600.
func writePrivate(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// partitionRoles maps each partition role to its current device path.
func partitionRoles(cfg *config.InstallConfig) map[string]string {
	if cfg.PartitionScheme == "manual" {
		roles := make(map[string]string, len(cfg.MountPoints))
		for mnt, part := range cfg.MountPoints {
			roles[mnt] = part
		}
		return roles
	}
	p := partSuffix(cfg.Disk)
	roles := map[string]string{"root": rootPartition(cfg)}
	if cfg.Firmware == "uefi" {
		roles["efi"] = cfg.Disk + p + "1"
	}
	if cfg.SwapMode == "partition" {
		roles["swap"] = cfg.Disk + p + "2"
	}
	return roles
}

func capturePartUUIDs(cfg *config.InstallConfig) (map[string]string, error) {
	uuids := make(map[string]string)
	for role, dev := range partitionRoles(cfg) {
		uuid, err := runOutput("blkid", "-s", "PARTUUID", "-o", "value", dev)
		if err != nil || uuid == "" {
			return nil, fmt.Errorf("read PARTUUID of %s: %v", dev, err)
		}
		uuids[role] = uuid
	}
	// Manual layouts key root by mount point; alias it so the ESP stub works.
	if root, ok := uuids["/"]; ok {
		uuids["root"] = root
	}
	return uuids, nil
}

// remapDevices points cfg at the partitions recorded in s.PartUUIDs, which may
// have different device names after a reboot, and verifies the layout matches.
func remapDevices(s *State) error {
	if len(s.PartUUIDs) == 0 {
		return nil
	}
	cfg := s.Config

	paths := make(map[string]string, len(s.PartUUIDs))
	for role, uuid := range s.PartUUIDs {
		path, err := filepath.EvalSymlinks("/dev/disk/by-partuuid/" + uuid)
		if err != nil {
			return fmt.Errorf("partition %q (PARTUUID %s) not found — is the install disk attached?", role, uuid)
		}
		paths[role] = path
	}

	disk, err := runOutput("lsblk", "-ndpo", "PKNAME", paths["root"])
	if err != nil || disk == "" {
		return fmt.Errorf("find disk of %s: %v", paths["root"], err)
	}
	cfg.Disk = disk

	if cfg.PartitionScheme == "manual" {
		for mnt := range cfg.MountPoints {
			if path, ok := paths[mnt]; ok {
				cfg.MountPoints[mnt] = path
			}
		}
		return nil
	}

	for role, want := range partitionRoles(cfg) {
		if paths[role] != want {
			return fmt.Errorf("partition layout on %s does not match the saved install (%s is %s, expected %s)",
				disk, role, paths[role], want)
		}
	}
	return nil
}

// PendingSteps returns the labels of the steps that still have to run.
func (s *State) PendingSteps() []string {
	var pending []string
	for _, def := range data.Steps(s.Config) {
		if !s.IsDone(def.Key) {
			pending = append(pending, def.Label)
		}
	}
	return pending
}

func parseLsblkPairs(out string) []map[string]string {
	var rows []map[string]string
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		row := make(map[string]string)
		for _, m := range lsblkPairRe.FindAllStringSubmatch(line, -1) {
			row[m[1]] = m[2]
		}
		rows = append(rows, row)
	}
	return rows
}
