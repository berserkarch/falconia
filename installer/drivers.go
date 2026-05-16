package installer

import (
	"fmt"
	"os"

	"falconia/config"
	"falconia/data"
)

// InstallDrivers installs hardware-specific packages based on the detected
// HardwareProfile: NVIDIA/Broadcom drivers and VM guest tools.
// Each section is a no-op if the relevant hardware was not detected.
func InstallDrivers(cfg *config.InstallConfig, log LineHandler) error {
	if err := installGPUDrivers(cfg, log); err != nil {
		return err
	}
	if err := installWiFiDrivers(cfg, log); err != nil {
		return err
	}
	return installVMTools(cfg, log)
}

func installGPUDrivers(cfg *config.InstallConfig, log LineHandler) error {
	hasNvidia := false
	hasIntel := false

	for _, gpu := range cfg.Hardware.GPUs {
		switch gpu.Vendor {
		case "nvidia":
			hasNvidia = true
			key := "nvidia"
			if gpu.UseOpen {
				key = "nvidia-open"
			}
			log(fmt.Sprintf("GPU: %s — installing %s", gpu.Model, key))
			pkgs := data.New().Add(data.ByDriver[key]...).Build()
			args := append([]string{"-S", "--noconfirm"}, pkgs...)
			if err := RunChrootDry(cfg, log, "pacman", args...); err != nil {
				return fmt.Errorf("install NVIDIA driver: %w", err)
			}
		case "intel":
			hasIntel = true
			log(fmt.Sprintf("GPU: %s — i915 built-in, no extra package needed", gpu.Model))
		case "amd":
			log(fmt.Sprintf("GPU: %s — amdgpu built-in, no extra package needed", gpu.Model))
		}
	}

	// Hybrid Intel + NVIDIA needs nvidia-prime for GPU switching
	if hasNvidia && hasIntel {
		log("Hybrid GPU detected — installing nvidia-prime")
		if err := RunChrootDry(cfg, log, "pacman", "-S", "--noconfirm", "nvidia-prime"); err != nil {
			log(fmt.Sprintf("Warning: nvidia-prime: %v", err))
		}
	}

	if hasNvidia {
		// Rebuild initramfs so the nvidia module is embedded
		log("Rebuilding initramfs for NVIDIA...")
		kver := detectKver()
		if kver == "" {
			kver = cfg.Kernel
		}
		outputPath := fmt.Sprintf("/boot/initramfs-%s.img", cfg.Kernel)
		if err := RunChrootDry(cfg, log, "dracut", "--force", outputPath, kver); err != nil {
			log(fmt.Sprintf("Warning: dracut rebuild after NVIDIA: %v", err))
		}
		// Regenerate GRUB config to pick up the new initramfs
		if cfg.Bootloader != "systemd-boot" {
			if err := RunChrootDry(cfg, log, "grub-mkconfig", "-o", "/boot/grub/grub.cfg"); err != nil {
				log(fmt.Sprintf("Warning: grub-mkconfig after NVIDIA: %v", err))
			}
		}
	}

	return nil
}

func installWiFiDrivers(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.Hardware.WiFi != "broadcom" {
		return nil
	}
	// LTS kernel needs the DKMS variant so the module rebuilds on kernel updates
	key := "broadcom"
	if cfg.Kernel == "linux-lts" {
		key = "broadcom-lts"
	}
	log(fmt.Sprintf("WiFi: Broadcom detected — installing %s", key))
	pkgs := data.ByDriver[key]
	args := append([]string{"-S", "--noconfirm"}, pkgs...)
	if err := RunChrootDry(cfg, log, "pacman", args...); err != nil {
		return fmt.Errorf("install Broadcom driver: %w", err)
	}
	return nil
}

func installVMTools(cfg *config.InstallConfig, log LineHandler) error {
	if cfg.Hardware.VM == "none" || cfg.Hardware.VM == "" {
		return nil
	}
	pkgs := data.ByVM[cfg.Hardware.VM]
	if len(pkgs) == 0 {
		return nil
	}
	log(fmt.Sprintf("VM: %s detected — installing guest tools", cfg.Hardware.VM))
	args := append([]string{"-S", "--noconfirm"}, pkgs...)
	if err := RunChrootDry(cfg, log, "pacman", args...); err != nil {
		return fmt.Errorf("install VM tools: %w", err)
	}
	// power-profiles-daemon is meaningless inside a VM
	if err := RunChrootDry(cfg, log, "pacman", "-R", "--noconfirm", "power-profiles-daemon"); err != nil {
		log(fmt.Sprintf("Warning: remove power-profiles-daemon: %v", err))
	}
	return nil
}

// detectKver returns the installed kernel version string from /mnt/lib/modules.
func detectKver() string {
	files, err := os.ReadDir("/mnt/lib/modules")
	if err == nil && len(files) > 0 {
		return files[0].Name()
	}
	return ""
}
