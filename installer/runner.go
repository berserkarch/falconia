package installer

import (
	"falconia/config"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// LineHandler is called for each line of combined stdout+stderr from a command.
type LineHandler func(line string)

// Run executes a command, calling handler for each output line.
// Returns an error if the command exits non-zero.
// handler may be nil (output is silently discarded).
func Run(handler LineHandler, name string, args ...string) error {
	if handler != nil {
		handler(fmt.Sprintf("$ %s %s", name, strings.Join(args, " ")))
	}

	cmd := exec.Command(name, args...)

	// Merge stdout and stderr into a single pipe.
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("pipe: %w", err)
	}
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return fmt.Errorf("start %s: %w", name, err)
	}

	// Close the write-end in this process so the reader gets EOF
	// when the child process exits.
	pw.Close()

	if handler != nil {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			handler(scanner.Text())
		}
	}
	pr.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// RunDry behaves like Run but skips execution if cfg.DryRun is true.
func RunDry(cfg *config.InstallConfig, handler LineHandler, name string, args ...string) error {
	if cfg.DryRun {
		if handler != nil {
			handler(styleGood("[DRY RUN] Would execute: ") + fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
			// Small delay to simulate work and let the user see the log
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}
	return Run(handler, name, args...)
}

func styleGood(s string) string {
	// Simple ANSI green for runner logs if not using lipgloss here
	return "\033[32m" + s + "\033[0m"
}

// RunChroot runs a command inside arch-chroot /mnt.
func RunChroot(handler LineHandler, name string, args ...string) error {
	chrootArgs := append([]string{"/mnt", name}, args...)
	return Run(handler, "arch-chroot", chrootArgs...)
}

// RunChrootDry behaves like RunChroot but respects DryRun.
func RunChrootDry(cfg *config.InstallConfig, handler LineHandler, name string, args ...string) error {
	if cfg.DryRun {
		if handler != nil {
			handler(styleGood("[DRY RUN] Would chroot execute: ") + fmt.Sprintf("%s %s", name, strings.Join(args, " ")))
			time.Sleep(100 * time.Millisecond)
		}
		return nil
	}
	return RunChroot(handler, name, args...)
}

// RunSh runs a shell command string (passed to /bin/sh -c).
func RunSh(handler LineHandler, script string) error {
	return Run(handler, "/bin/sh", "-c", script)
}

// RunChrootSh runs a shell script string inside arch-chroot /mnt.
func RunChrootSh(handler LineHandler, script string) error {
	return RunChroot(handler, "/bin/sh", "-c", script)
}
