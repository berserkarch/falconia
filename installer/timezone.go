package installer

import (
	"os"
	"path/filepath"
	"strings"
)

// ListTimezones returns a list of system timezones.
// posix/ and right/ subtrees are skipped — they contain duplicates of the
// top-level zones under alternate naming conventions.
func ListTimezones() []string {
	var zones []string
	root := "/usr/share/zoneinfo"

	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		// Skip POSIX/right duplicate trees
		if strings.HasPrefix(rel, "posix/") || strings.HasPrefix(rel, "right/") {
			return nil
		}

		// Keep Region/City and a few useful top-level zones
		if strings.Contains(rel, "/") {
			zones = append(zones, rel)
		} else if rel == "UTC" {
			zones = append(zones, rel)
		}

		return nil
	})

	return zones
}
