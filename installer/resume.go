package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"falconia/data"
)

const probeDir = stateDir + "/probe"

var lsblkPairRe = regexp.MustCompile(`([A-Z-]+)="([^"]*)"`)

// PasswordPrompt asks the user for the LUKS passphrase of dev.
// An empty answer skips that partition.
type PasswordPrompt func(dev string) (string, error)

// FindResumeState locates the saved state of an interrupted install.
//
// It first checks /tmp (same live session). After a reboot /tmp is empty, so
// it scans partitions: an ESP stub names the root partition (and holds the
// LUKS passphrase); without one, each root-capable partition is probed, asking
// for a passphrase for LUKS containers. Found state is copied back to /tmp.
//
// The partitions recorded in the state are then matched by PARTUUID so a disk
// that was renamed (sda -> sdb) is still found, and a different disk is refused.
func FindResumeState(dryRun bool, ask PasswordPrompt, log LineHandler) (*State, error) {
	s, err := loadStateFile(localStatePath(dryRun))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if s == nil {
		if dryRun {
			return nil, fmt.Errorf("no dry-run install to resume (%s not found)", dryRunStatePath)
		}
		if s, err = scanForState(ask, log); err != nil {
			return nil, err
		}
	}

	if s.Config.DryRun != dryRun {
		if s.Config.DryRun {
			return nil, fmt.Errorf("saved state is from a --dry-run install")
		}
		return nil, fmt.Errorf("saved state is from a real install; resume it without --dry-run")
	}
	if !dryRun {
		if busy := runningInstallTools(); len(busy) > 0 {
			return nil, fmt.Errorf("%s from the previous run is still running; wait for it to exit (or kill it) first",
				strings.Join(busy, ", "))
		}
		if err := remapDevices(s); err != nil {
			return nil, err
		}
		if err := SaveState(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// runningInstallTools lists install commands that are still running, e.g.
// orphaned when the installer was quit mid-step. Resuming alongside them would
// run two package transactions against the same target.
func runningInstallTools() []string {
	var busy []string
	for _, name := range []string{"pacstrap", "pacman", "arch-chroot", "dracut", "grub-install", "cryptsetup", "mkfs.btrfs", "mkfs.ext4", "mkfs.xfs"} {
		if exec.Command("pgrep", "-x", name).Run() == nil {
			busy = append(busy, name)
		}
	}
	return busy
}

func loadStateFile(path string) (*State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Version != StateVersion || s.Config == nil {
		return nil, fmt.Errorf("%s was written by an incompatible installer version", path)
	}
	return &s, nil
}

type blockDev struct {
	path, fstype, partuuid string
}

func listPartitions() ([]blockDev, error) {
	out, err := runOutput("lsblk", "-Pnpo", "PATH,FSTYPE,PARTUUID,TYPE")
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	var parts []blockDev
	for _, row := range parseLsblkPairs(out) {
		if row["TYPE"] == "part" {
			parts = append(parts, blockDev{row["PATH"], row["FSTYPE"], row["PARTUUID"]})
		}
	}
	return parts, nil
}

func scanForState(ask PasswordPrompt, log LineHandler) (*State, error) {
	parts, err := listPartitions()
	if err != nil {
		return nil, err
	}

	// An ESP stub pins down the root partition directly.
	for _, p := range parts {
		if p.fstype != "vfat" {
			continue
		}
		stub, err := readESPStub(p.path)
		if err != nil || stub == nil {
			continue
		}
		for _, root := range parts {
			if root.partuuid == stub.RootPartUUID {
				log("Found resume data on " + p.path + ", root is " + root.path)
				return probeRoot(root, stub.EncryptionPass, ask, log)
			}
		}
	}

	// No stub (BIOS install, or the ESP wasn't mounted yet): try every candidate.
	for _, p := range parts {
		switch p.fstype {
		case "ext4", "btrfs", "xfs", "crypto_LUKS":
		default:
			continue
		}
		if s, err := probeRoot(p, "", ask, log); err == nil {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no interrupted install found (checked %s and all partitions)", tmpStatePath)
}

func readESPStub(dev string) (*espStub, error) {
	if err := mountProbe(dev, "ro"); err != nil {
		return nil, err
	}
	defer unmountProbe()

	b, err := os.ReadFile(filepath.Join(probeDir, espStubName))
	if err != nil {
		return nil, nil
	}
	var stub espStub
	if err := json.Unmarshal(b, &stub); err != nil || stub.Version != StateVersion {
		return nil, nil
	}
	return &stub, nil
}

// probeRoot opens (if needed) and mounts p read-only to read the saved state.
// An opened LUKS container is left open for RestoreEnvironment to reuse.
func probeRoot(p blockDev, pass string, ask PasswordPrompt, log LineHandler) (*State, error) {
	dev := p.path
	opened := false
	if p.fstype == "crypto_LUKS" {
		if fileExists(cryptrootMapper) {
			return nil, fmt.Errorf("%s is already open; close it or resume from the same session", cryptrootMapper)
		}
		if pass == "" {
			if ask == nil {
				return nil, fmt.Errorf("%s is encrypted", dev)
			}
			var err error
			if pass, err = ask(dev); err != nil {
				return nil, err
			}
			if pass == "" {
				return nil, fmt.Errorf("skipped %s", dev)
			}
		}
		if err := runWithStdin(nil, strings.NewReader(pass), "cryptsetup", "open", dev, "cryptroot", "-d", "-"); err != nil {
			log("Could not unlock " + dev)
			return nil, err
		}
		opened = true
		dev = cryptrootMapper
	}

	s, err := readRootState(dev)
	if err != nil {
		if opened {
			_ = Run(nil, "cryptsetup", "close", "cryptroot")
		}
		return nil, err
	}
	if s.Config.EncryptDisk && s.Config.EncryptionPass == "" {
		s.Config.EncryptionPass = pass
	}
	return s, nil
}

func readRootState(dev string) (*State, error) {
	// Guided btrfs keeps the root filesystem in the @ subvolume.
	for _, opts := range []string{"ro,subvol=/@", "ro"} {
		if mountProbe(dev, opts) != nil {
			continue
		}
		s, err := loadStateFile(filepath.Join(probeDir, diskStateFile))
		unmountProbe()
		if err == nil {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no resume data on %s", dev)
}

func mountProbe(dev, opts string) error {
	if err := os.MkdirAll(probeDir, 0o700); err != nil {
		return err
	}
	return Run(nil, "mount", "-o", opts, dev, probeDir)
}

func unmountProbe() {
	_ = Run(nil, "umount", probeDir)
}

// RestoreEnvironment brings the live system back to where the interrupted run
// left off: LUKS open, filesystems mounted, swap on. Steps that ran before the
// disk was touched need nothing restored.
func RestoreEnvironment(s *State, log LineHandler) error {
	cfg := s.Config
	if !s.IsDone(data.StepFormatDisks) {
		log("Disk was not formatted yet; nothing to restore")
		return nil
	}
	if cfg.EncryptDisk {
		if err := OpenLuks(cfg, log); err != nil {
			return fmt.Errorf("open LUKS: %w", err)
		}
	}
	if !s.IsDone(data.StepMountDisks) {
		return nil
	}
	if err := MountDisks(cfg, log); err != nil {
		return err
	}
	if s.IsDone(data.StepSetupSwap) {
		if err := swaponOnce(cfg, log, "/mnt/swapfile"); err != nil {
			log("Warning: swapon /mnt/swapfile: " + err.Error())
		}
	}
	// An interrupted pacman transaction leaves its lock behind.
	if !cfg.DryRun {
		if err := removeIfExists(log, "/mnt/var/lib/pacman/db.lck"); err != nil {
			return err
		}
	}
	return nil
}
