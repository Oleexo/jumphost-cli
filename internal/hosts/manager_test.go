package hosts

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyAndRestore(t *testing.T) {
	tmpDir := t.TempDir()
	hostsPath := filepath.Join(tmpDir, "hosts")
	backupPath := filepath.Join(tmpDir, "hosts.bak")
	orig := "127.0.0.1\tlocalhost\n"
	require.NoError(t, os.WriteFile(hostsPath, []byte(orig), 0644))
	m := &Manager{Path: hostsPath, BackupPath: backupPath, marker: "# test"}
	modified, err := m.Apply("db.example")
	require.NoError(t, err)
	require.True(t, modified)
	data, _ := os.ReadFile(hostsPath)
	require.Contains(t, string(data), "db.example")
	// restore
	require.NoError(t, m.Restore())
	data, _ = os.ReadFile(hostsPath)
	require.Equal(t, orig, string(data))
}

func TestApplyIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	hostsPath := filepath.Join(tmpDir, "hosts2")
	backupPath := filepath.Join(tmpDir, "hosts2.bak")
	orig := "127.0.0.1\tlocalhost\n"
	require.NoError(t, os.WriteFile(hostsPath, []byte(orig), 0644))
	m := &Manager{Path: hostsPath, BackupPath: backupPath, marker: "# test2"}
	modified, err := m.Apply("db.example")
	require.NoError(t, err)
	require.True(t, modified)
	modified, err = m.Apply("db.example")
	require.NoError(t, err)
	require.False(t, modified, "second apply should be no-op")
}

func TestFilterLines(t *testing.T) {
	input := "line1\nkeep this\nline2 # marker\n"
	r := bufio.NewReader(strings.NewReader(input))
	lines, err := FilterLines(r, "# marker")
	require.NoError(t, err)
	joined := strings.Join(lines, "\n")
	require.NotContains(t, joined, "# marker")
	require.Contains(t, joined, "keep this")
}

func TestRestoreNoBackup(t *testing.T) {
	m := &Manager{
		Path: filepath.Join(t.TempDir(), "hosts"), BackupPath: filepath.Join(t.TempDir(), "bak"), marker: "# m",
	}
	// no backup created -> should be no error
	require.NoError(t, m.Restore())
}
