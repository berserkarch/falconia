package installer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// readFile reads a file and returns its contents as a string.
func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

// writeChroot writes content to a path on the host (which maps inside /mnt).
func writeChroot(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// appendFile appends content to a file, creating it if needed.
func appendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// runOutput runs a command and returns combined output as a trimmed string.
func runOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w\n%s", name, err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// runWithStdin runs a command with the given stdin reader.
func runWithStdin(log LineHandler, stdin io.Reader, name string, args ...string) error {
	if log != nil {
		log(fmt.Sprintf("$ %s %s", name, strings.Join(args, " ")))
	}

	var buf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Run(); err != nil {
		if log != nil {
			log(buf.String())
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	if log != nil && buf.Len() > 0 {
		log(buf.String())
	}
	return nil
}

// contains checks if s contains substr (case-sensitive).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
