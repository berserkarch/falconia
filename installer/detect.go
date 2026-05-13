package installer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	out, err := exec.Command("lsblk", "-d", "-o", "PATH,MODEL,SIZE", "--noheadings", "--json").Output()
	if err != nil {
		// fallback: parse plain text
		return listDisksFallback()
	}
	// simple line parser (avoid importing encoding/json for brevity)
	var disks []DiskInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, `"path"`) {
			// we'll just use the fallback parser; JSON parsing without import is fragile
			break
		}
	}
	if len(disks) == 0 {
		return listDisksFallback()
	}
	return disks, nil
}

func listDisksFallback() ([]DiskInfo, error) {
	out, err := exec.Command("lsblk", "-d", "-o", "PATH,MODEL,SIZE", "--noheadings").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	var disks []DiskInfo
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 1 {
			continue
		}
		path := fields[0]
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

// CheckInternet returns nil if archlinux.org is reachable.
func CheckInternet(log LineHandler) error {
	return Run(log, "ping", "-c", "1", "-W", "3", "archlinux.org")
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
