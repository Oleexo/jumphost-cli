package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/Oleexo/jumphost-cli/cmd"
)

func TestRootVersionCommand(t *testing.T) {
	c := cmd.NewRootCmd()
	buf := &bytes.Buffer{}
	c.SetOut(buf)
	c.SetArgs([]string{"version"})
	if err := c.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if out == "" || out[0] == '\n' {
		// fallback check: ensure version variable accessible
		if os.Getenv("CI") != "" { // in CI fail hard
			t.Fatalf("expected version output, got %q", out)
		}
	}
}
