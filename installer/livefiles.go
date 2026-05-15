package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"falconia/config"
)

// liveFiles lists absolute paths that are copied verbatim from the running
// live environment into the new installation at /mnt.
// Add entries here as new requirements arise.
var liveFiles = []string{
	"/etc/pacman.conf",
	"/etc/dracut.conf.d/berserkarch.conf",
}

// CopyLiveFiles copies each path in liveFiles from the live environment into
// /mnt, creating intermediate directories as needed.
func CopyLiveFiles(cfg *config.InstallConfig, log LineHandler) error {
	for _, path := range liveFiles {
		dst := "/mnt" + path
		if cfg.DryRun {
			log(styleGood("[DRY RUN] Would copy: ") + path + " → " + dst)
			continue
		}
		if err := copyLiveFile(log, path, dst); err != nil {
			return fmt.Errorf("copy %s: %w", path, err)
		}
	}
	return nil
}

func copyLiveFile(log LineHandler, src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	log(fmt.Sprintf("$ cp %s %s", src, dst))
	return nil
}
