package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartPortForwarding_NoAws(t *testing.T) {
	_ = os.Setenv("JUMPHOST_DISABLE_PTY", "1")
	oldPath := os.Getenv("PATH")
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("PATH", oldPath)
	_ = os.Setenv("PATH", "")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := StartPortForwarding(ctx, "i-123", "db.example", 5432, 15432, "us-east-1")
	if err == nil || !strings.Contains(err.Error(), "aws cli not found") {
		to := "<nil>"
		if err != nil {
			to = err.Error()
		}
		t.Fatalf("expected aws not found error, got %s", to)
	}
}

func TestStartPortForwarding_NoPlugin(t *testing.T) {
	_ = os.Setenv("JUMPHOST_DISABLE_PTY", "1")
	// create fake aws but no plugin
	tmp := t.TempDir()
	fakeAws := filepath.Join(tmp, "aws")
	if err := os.WriteFile(fakeAws,
		[]byte("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'aws-cli/2.15.0'; exit 0; fi\nexit 0\n"),
		0755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("PATH", oldPath)
	_ = os.Setenv("PATH", tmp)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := StartPortForwarding(ctx, "i-xyz", "db.example", 5432, 15432, "us-east-1")
	if err == nil || !strings.Contains(err.Error(), "session-manager-plugin not found") {
		t.Fatalf("expected plugin not found error, got %v", err)
	}
}

func TestStartPortForwarding_SuccessFakeAws(t *testing.T) {
	_ = os.Setenv("JUMPHOST_DISABLE_PTY", "1")
	// include fake plugin
	tmp := t.TempDir()
	fakeAws := filepath.Join(tmp, "aws")
	logFile := filepath.Join(tmp, "args.txt")
	awsScript := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'aws-cli/2.15.0'; exit 0; fi\necho \"$@\" > '" + logFile + "'\nexit 0\n"
	if err := os.WriteFile(fakeAws, []byte(awsScript), 0755); err != nil {
		t.Fatal(err)
	}
	fakePlugin := filepath.Join(tmp, "session-manager-plugin")
	pluginScript := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '1.2.398.0'; exit 0; fi\nexit 0\n"
	if err := os.WriteFile(fakePlugin, []byte(pluginScript), 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	defer func(key, value string) {
		_ = os.Setenv(key, value)
	}("PATH", oldPath)
	_ = os.Setenv("PATH", tmp)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := StartPortForwarding(ctx, "i-abc", "db.example", 5432, 15432, "us-west-2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("missing log file: %v", err)
	}
	out := string(data)
	if !strings.Contains(out, "start-session") || !strings.Contains(out, "i-abc") || !strings.Contains(out,
		"us-west-2") {
		t.Fatalf("unexpected aws args captured: %s", out)
	}
}
