package hosts

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Manager manages temporary redirection in hosts file.
type Manager struct {
	Path       string
	BackupPath string
	marker     string
}

func NewDefaultManager() *Manager {
	return &Manager{
		Path: "/etc/hosts", BackupPath: filepath.Join(os.TempDir(), "hosts.backup.jumphost"), marker: "# jumphost-temp",
	}
}

// Apply adds an entry mapping host to 127.0.0.1 if not already present. Returns true if modified.
func (m *Manager) Apply(host string) (bool, error) {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), host) && strings.Contains(string(data), m.marker) {
		return false, nil // already applied
	}
	if err := os.WriteFile(m.BackupPath, data, 0600); err != nil {
		return false, err
	}
	f, err := os.OpenFile(m.Path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	line := fmt.Sprintf("127.0.0.1\t%s\t%s\n", host, m.marker)
	if _, err = f.WriteString(line); err != nil {
		return false, err
	}
	return true, nil
}

// Restore restores the hosts file from backup if backup exists / contains marker entry.
func (m *Manager) Restore() error {
	bak, err := os.ReadFile(m.BackupPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// crude verification that it's the right backup
	if !strings.Contains(string(bak), "127.0.0.1") {
		return errors.New("backup invalid")
	}
	return os.WriteFile(m.Path, bak, 0644)
}

// FilterLines removes lines containing the marker from input (used in tests / potential cleanup).
func FilterLines(r io.Reader, marker string) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		l := scanner.Text()
		if !strings.Contains(l, marker) {
			lines = append(lines, l)
		}
	}
	return lines, scanner.Err()
}
