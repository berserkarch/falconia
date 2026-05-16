package installer

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"falconia/config"
)

// DetectHardware runs all hardware probes and returns a populated profile.
// Called once in main.go before the TUI starts; results stored in cfg.Hardware.
func DetectHardware() config.HardwareProfile {
	return config.HardwareProfile{
		CPU:      detectCPU(),
		GPUs:     detectGPUs(),
		WiFi:     detectWiFi(),
		VM:       detectVM(),
		RAMBytes: detectRAM(),
		HasNVMe:  detectNVMe(),
	}
}

// detectCPU returns "intel", "amd", or "other" by reading /proc/cpuinfo.
func detectCPU() string {
	data, _ := os.ReadFile("/proc/cpuinfo")
	switch {
	case strings.Contains(string(data), "GenuineIntel"):
		return "intel"
	case strings.Contains(string(data), "AuthenticAMD"):
		return "amd"
	default:
		return "other"
	}
}

// detectGPUs parses lspci -nn for display-class PCI devices (class 03xx).
// For NVIDIA GPUs it determines whether nvidia-open is appropriate based on
// the device ID: Turing and later (roughly device ID >= 0x1e00) are supported.
func detectGPUs() []config.GPUInfo {
	out, err := exec.Command("lspci", "-nn").Output()
	if err != nil {
		return nil
	}

	var gpus []config.GPUInfo
	for _, line := range strings.Split(string(out), "\n") {
		// PCI display class codes: 0300 VGA, 0302 3D controller, 0380 Display
		if !strings.Contains(line, "[0300]") &&
			!strings.Contains(line, "[0302]") &&
			!strings.Contains(line, "[0380]") {
			continue
		}

		gpu := config.GPUInfo{Model: extractGPUModel(line)}

		switch {
		case strings.Contains(line, "[10de:"):
			gpu.Vendor = "nvidia"
			// Extract device ID and check for Turing+ (>= 0x1e00)
			if id := extractPCIDeviceID(line, "10de"); id != "" {
				if n, err := strconv.ParseInt(id, 16, 32); err == nil && n >= 0x1e00 {
					gpu.UseOpen = true
				}
			}
		case strings.Contains(line, "[1002:"):
			gpu.Vendor = "amd"
		case strings.Contains(line, "[8086:"):
			gpu.Vendor = "intel"
		default:
			gpu.Vendor = "other"
		}

		gpus = append(gpus, gpu)
	}
	return gpus
}

// extractGPUModel pulls the human-readable device name from a lspci -nn line.
// Example line:
//
//	01:00.0 VGA compatible controller [0300]: NVIDIA Corporation TU116 [GeForce GTX 1660] [10de:2182] (rev a1)
//
// Returns "NVIDIA Corporation TU116 [GeForce GTX 1660]".
func extractGPUModel(line string) string {
	// Everything after "]: " is "Vendor Device [pci-id] (rev)"
	idx := strings.Index(line, "]: ")
	if idx < 0 {
		return ""
	}
	rest := line[idx+3:]
	// Strip trailing PCI ID bracket and revision
	if end := strings.LastIndex(rest, " ["); end > 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

// extractPCIDeviceID finds "[VENDOR:DEVICE]" in a lspci -nn line and returns
// the four-character device hex string.
func extractPCIDeviceID(line, vendor string) string {
	prefix := "[" + vendor + ":"
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(prefix):]
	end := strings.Index(rest, "]")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// detectWiFi returns "broadcom" if a Broadcom wireless adapter is found via
// lspci, "other" otherwise. We check for PCI class [0280] (Network controller)
// to avoid false-positives from Broadcom Ethernet adapters (class [0200])
// which share the same vendor ID (14e4) but use in-tree drivers, not broadcom-wl.
func detectWiFi() string {
	out, err := exec.Command("lspci", "-nn").Output()
	if err != nil {
		return "other"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "[14e4:") && strings.Contains(line, "[0280]") {
			return "broadcom"
		}
	}
	return "other"
}

// detectVM uses systemd-detect-virt to identify the virtualisation platform.
// Returns "virtualbox", "vmware", "kvm", "qemu", or "none".
func detectVM() string {
	out, err := exec.Command("systemd-detect-virt").Output()
	if err != nil {
		return "none"
	}
	switch strings.TrimSpace(string(out)) {
	case "oracle":
		return "virtualbox"
	case "vmware":
		return "vmware"
	case "kvm":
		return "kvm"
	case "qemu":
		return "qemu"
	default:
		return "none"
	}
}

// detectRAM returns total physical RAM in bytes by reading /proc/meminfo.
func detectRAM() int64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err == nil {
			return kb * 1024
		}
	}
	return 0
}

// detectNVMe returns true if any /dev/nvme* device node exists.
func detectNVMe() bool {
	entries, err := os.ReadDir("/dev")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "nvme") {
			return true
		}
	}
	return false
}
