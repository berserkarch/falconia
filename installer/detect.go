package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DiskInfo holds info about a block device shown in the disk picker.
type DiskInfo struct {
	Path  string // e.g. "/dev/sda"
	Model string // e.g. "Samsung SSD 870 EVO"
	Size  string // e.g. "500G"
}

// DetectFirmware returns "uefi" if /sys/firmware/efi exists, else "bios".
func DetectFirmware() string {
	if _, err := os.Stat("/sys/firmware/efi"); err == nil {
		return "uefi"
	}
	return "bios"
}

// ListDisks returns all physical block devices (type "disk") via lsblk.
func ListDisks() ([]DiskInfo, error) {
	return listDisksFallback()
}

func listDisksFallback() ([]DiskInfo, error) {
	// -e 7: exclude loop devices (major 7); -e 11: exclude optical drives (major 11)
	out, err := exec.Command("lsblk", "-d", "-e", "7,11", "-o", "PATH,MODEL,SIZE", "--noheadings").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	liveISO := liveIsoDisk()
	var disks []DiskInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 1 {
			continue
		}
		path := fields[0]
		if liveISO != "" && path == liveISO {
			continue
		}
		size := ""
		model := ""
		if len(fields) >= 2 {
			size = fields[len(fields)-1]
			if len(fields) >= 3 {
				model = strings.Join(fields[1:len(fields)-1], " ")
			}
		}
		disks = append(disks, DiskInfo{
			Path:  path,
			Model: model,
			Size:  size,
		})
	}
	if len(disks) == 0 {
		return nil, fmt.Errorf("no block devices found")
	}
	return disks, nil
}

// liveIsoDisk returns the parent disk path of the Arch ISO boot medium by
// reading /proc/mounts for the /run/archiso/bootmnt mount point.
// Returns "" if the mount point isn't found or the parent can't be determined.
func liveIsoDisk() string {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "/run/archiso/bootmnt" {
			continue
		}
		// fields[0] is e.g. /dev/sda1 — get the parent disk name via lsblk
		out, err := exec.Command("lsblk", "-n", "-o", "PKNAME", fields[0]).Output()
		if err != nil {
			return ""
		}
		pkname := strings.TrimSpace(string(out))
		if pkname == "" {
			return ""
		}
		return "/dev/" + pkname
	}
	return ""
}

// CheckInternet returns nil if an internet connection is detected.
// Retries up to 3 times with a 3-second timeout each to tolerate transient blips.
func CheckInternet(log LineHandler) error {
	const attempts = 3
	for i := 1; i <= attempts; i++ {
		if i > 1 {
			log(fmt.Sprintf("No response, retrying (%d/%d)...", i, attempts))
			time.Sleep(2 * time.Second)
		}
		if err := Run(log, "ping", "-c", "1", "-W", "3", "1.1.1.1"); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no internet connection after %d attempts — check your network", attempts)
}

// SyncClock syncs the system clock via timedatectl.
func SyncClock(log LineHandler) error {
	return Run(log, "timedatectl", "set-ntp", "true")
}

// RebootCmd returns a bubbletea Cmd that reboots the system.
// If dryRun is true, it returns tea.Quit instead of rebooting.
func RebootCmd(dryRun bool) tea.Cmd {
	if dryRun {
		return tea.Quit
	}
	return func() tea.Msg {
		_ = exec.Command("reboot").Run()
		return nil
	}
}

// PartitionInfo holds info about a partition.
type PartitionInfo struct {
	Path string // e.g. "/dev/sda1"
	Size string // e.g. "512M"
	Type string // e.g. "vfat" or "part"
}

// ListPartitions returns all partitions for a given disk.
func ListPartitions(disk string) ([]PartitionInfo, error) {
	out, err := exec.Command("lsblk", "-n", "-o", "PATH,SIZE,TYPE", disk).Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}

	var parts []PartitionInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		// Skip the disk itself (type "disk")
		if fields[2] != "part" {
			continue
		}
		parts = append(parts, PartitionInfo{
			Path: fields[0],
			Size: fields[1],
			Type: fields[2],
		})
	}
	return parts, nil
}
