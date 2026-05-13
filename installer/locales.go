package installer

import (
	"bufio"
	"os"
	"strings"
)

// ListLocales returns a list of available locales from /etc/locale.gen.
func ListLocales() []string {
	var locales []string
	f, err := os.Open("/etc/locale.gen")
	if err != nil {
		// Fallback to a very minimal list if file doesn't exist
		return []string{"en_US.UTF-8"}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Locales start with a letter (uncommented) or # followed by a letter
		if strings.HasPrefix(line, "#") {
			line = strings.TrimPrefix(line, "#")
			line = strings.TrimSpace(line)
		}

		if len(line) > 0 && (line[0] >= 'a' && line[0] <= 'z') {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				locales = append(locales, parts[0])
			}
		}
	}

	// Remove duplicates and sort if needed, but locale.gen is usually sorted
	return uniqueStrings(locales)
}

func uniqueStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}
	return u
}
