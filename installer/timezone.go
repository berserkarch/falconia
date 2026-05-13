package installer

import (
	"os"
	"path/filepath"
	"strings"
)

// ListTimezones returns a list of system timezones.
func ListTimezones() []string {
	var zones []string
	root := "/usr/share/zoneinfo"

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		// Filter out internal/legacy paths
		if strings.HasPrefix(rel, "posix/") || strings.HasPrefix(rel, "right/") || strings.HasPrefix(rel, "Etc/") {
			// Actually keep Etc/ for UTC etc
		} else if strings.Count(rel, "/") == 0 {
			// Skip top-level files like "EST", "GMT" if we want strictly Region/City
			// But some like "UTC" are useful. Let's keep them if they are common.
		}

		// Basic filter for Region/City
		if strings.Contains(rel, "/") {
			zones = append(zones, rel)
		} else if rel == "UTC" {
			zones = append(zones, rel)
		}

		return nil
	})

	return zones
}
