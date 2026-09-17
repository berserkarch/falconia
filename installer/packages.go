package installer

import (
	"slices"
	"strings"

	"falconia/config"
)

// withoutExcluded drops every package listed in cfg.ExcludePackages from pkgs,
// logging what was skipped. Only packages the installer names explicitly are
// affected; pacman still pulls in excluded packages that others depend on.
func withoutExcluded(cfg *config.InstallConfig, log LineHandler, pkgs []string) []string {
	if len(cfg.ExcludePackages) == 0 {
		return pkgs
	}
	kept := make([]string, 0, len(pkgs))
	var skipped []string
	for _, pkg := range pkgs {
		if slices.Contains(cfg.ExcludePackages, pkg) {
			skipped = append(skipped, pkg)
		} else {
			kept = append(kept, pkg)
		}
	}
	if len(skipped) > 0 {
		log(styleGood("[EXCLUDE] ") + "skipping " + strings.Join(skipped, " "))
	}
	return kept
}

// pacmanInstall installs pkgs inside the chroot, minus excluded packages.
// It is a no-op when nothing is left to install.
func pacmanInstall(cfg *config.InstallConfig, log LineHandler, pkgs ...string) error {
	pkgs = withoutExcluded(cfg, log, pkgs)
	if len(pkgs) == 0 {
		return nil
	}
	args := append([]string{"-S", "--noconfirm", "--needed"}, pkgs...)
	return RunChrootDry(cfg, log, "pacman", args...)
}

// MergeExcludes adds pkgs to cfg.ExcludePackages, skipping duplicates.
func MergeExcludes(cfg *config.InstallConfig, pkgs []string) {
	for _, pkg := range pkgs {
		if pkg != "" && !slices.Contains(cfg.ExcludePackages, pkg) {
			cfg.ExcludePackages = append(cfg.ExcludePackages, pkg)
		}
	}
}
