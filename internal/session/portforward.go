package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/mattn/go-isatty"
)

// StartPortForwarding invokes aws ssm start-session using the AWS CLI (keeps parity with bash script).
func StartPortForwarding(ctx context.Context, instanceID, host string, remotePort, localPort int, region string) error {
	// New: preflight version / dependency checks
	if err := preflightAWSDeps(ctx); err != nil { // returns rich error if missing
		return err
	}

	if os.Getenv("JUMPHOST_DISABLE_PTY") == "1" {
		// Force direct path only; do not attempt PTY or script fallback.
		args := []string{
			"ssm", "start-session", "--target", instanceID, "--document-name",
			"AWS-StartPortForwardingSessionToRemoteHost",
			"--parameters",
			fmt.Sprintf("{\"host\":[\"%s\"],\"portNumber\":[\"%d\"],\"localPortNumber\":[\"%d\"]}", host, remotePort,
				localPort), "--region", region,
		}
		if err := runDirect(ctx, args, os.Getenv("JUMPHOST_DEBUG") == "1"); err != nil {
			// Check for graceful cancellation
			if errors.Is(err, context.Canceled) {
				return context.Canceled
			}
			return finalizeErr(err)
		}
		return nil
	}

	params := fmt.Sprintf("{\"host\":[\"%s\"],\"portNumber\":[\"%d\"],\"localPortNumber\":[\"%d\"]}", host, remotePort,
		localPort)
	args := []string{
		"ssm", "start-session", "--target", instanceID, "--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters", params, "--region", region,
	}

	debug := os.Getenv("JUMPHOST_DEBUG") == "1"

	usePTY := false
	if os.Getenv("JUMPHOST_FORCE_PTY") == "1" || os.Getenv("GOLAND_IDE") != "" { // JetBrains sets GOLAND_IDE
		usePTY = true
		if debug {
			_, _ = fmt.Fprintln(os.Stderr, "[jumphost] forcing PTY due to env")
		}
	}

	if !usePTY {
		stdinIsTTY := isatty.IsTerminal(os.Stdin.Fd())
		stdoutIsTTY := isatty.IsTerminal(os.Stdout.Fd())
		stderrIsTTY := isatty.IsTerminal(os.Stderr.Fd())
		if debug {
			_, _ = fmt.Fprintf(os.Stderr, "[jumphost] tty states stdin=%v stdout=%v stderr=%v\n", stdinIsTTY,
				stdoutIsTTY,
				stderrIsTTY)
		}
		// Only attempt direct if *all* are terminals (otherwise we know we need a PTY).
		if !stdinIsTTY || !stdoutIsTTY || !stderrIsTTY {
			usePTY = true
			if debug {
				_, _ = fmt.Fprintln(os.Stderr, "[jumphost] selecting PTY because not all std fds are terminals")
			}
		}
	}

	if !usePTY { // try direct, fallback if needed
		if debug {
			_, _ = fmt.Fprintln(os.Stderr, "[jumphost] attempting direct execution of aws cli")
		}
		if err := runDirect(ctx, args, debug); err != nil {
			// Check for graceful cancellation
			if errors.Is(err, context.Canceled) {
				return context.Canceled
			}
			if isTTYError(err) {
				if debug {
					_, _ = fmt.Fprintf(os.Stderr, "[jumphost] direct attempt hit TTY error, retrying with PTY: %v\n",
						err)
				}
				if err2 := runWithPTY(ctx, args, debug); err2 != nil {
					// Check for graceful cancellation
					if errors.Is(err2, context.Canceled) {
						return context.Canceled
					}
					if isTTYError(err2) {
						if debug {
							_, _ = fmt.Fprintf(os.Stderr,
								"[jumphost] PTY attempt still TTY error, trying 'script' fallback: %v\n", err2)
						}
						err3 := runWithScript(ctx, args, debug)
						// Check for graceful cancellation
						if errors.Is(err3, context.Canceled) {
							return context.Canceled
						}
						return finalizeErr(err3)
					}
					return finalizeErr(err2)
				}
				return nil
			}
			return finalizeErr(err)
		}
		return nil
	}

	// PTY path first
	if debug {
		_, _ = fmt.Fprintln(os.Stderr, "[jumphost] starting with PTY execution of aws cli")
	}
	if err := runWithPTY(ctx, args, debug); err != nil {
		// Check for graceful cancellation
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		if isTTYError(err) {
			if debug {
				_, _ = fmt.Fprintf(os.Stderr, "[jumphost] PTY attempt TTY error, trying 'script' fallback: %v\n", err)
			}
			err2 := runWithScript(ctx, args, debug)
			// Check for graceful cancellation
			if errors.Is(err2, context.Canceled) {
				return context.Canceled
			}
			return finalizeErr(err2)
		}
		return finalizeErr(err)
	}
	return nil
}

func runDirect(ctx context.Context, args []string, debug bool) error {
	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	var stderrBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	if err := cmd.Run(); err != nil {
		// Check if the context was cancelled (CTRL+C)
		if ctx.Err() == context.Canceled {
			return context.Canceled
		}
		return fmt.Errorf("aws cli direct exec failed: %w | stderr: %s", err, strings.TrimSpace(stderrBuf.String()))
	}
	if debug {
		_, _ = fmt.Fprintln(os.Stderr, "[jumphost] direct execution succeeded")
	}
	return nil
}

func runWithPTY(ctx context.Context, args []string, debug bool) error {
	cmd := exec.CommandContext(ctx, "aws", args...)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("start pty for aws cli: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// tee PTY output to stdout and buffer
		_, _ = io.Copy(io.MultiWriter(os.Stdout, &buf), ptmx)
	}()

	if isatty.IsTerminal(os.Stdin.Fd()) { // optional input forward
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	}

	err = cmd.Wait()
	wg.Wait()
	if err != nil {
		// Check if the context was cancelled (CTRL+C)
		if ctx.Err() == context.Canceled {
			return context.Canceled
		}
		return errors.Join(fmt.Errorf("aws cli pty exec failed: %w | output: %s", err, truncate(buf.String(), 4000)))
	}
	if debug {
		_, _ = fmt.Fprintln(os.Stderr, "[jumphost] PTY execution succeeded")
	}
	return nil
}

func runWithScript(ctx context.Context, args []string, debug bool) error {
	if _, err := exec.LookPath("script"); err != nil {
		return fmt.Errorf("'script' command not found for fallback: %w", err)
	}
	fullArgs := append([]string{"-q", "/dev/null", "aws"}, args...)
	cmd := exec.CommandContext(ctx, "script", fullArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if debug {
		_, _ = fmt.Fprintln(os.Stderr, "[jumphost] attempting 'script' fallback wrapper")
	}
	if err := cmd.Run(); err != nil {
		// Check if the context was cancelled (CTRL+C)
		if ctx.Err() == context.Canceled {
			return context.Canceled
		}
		return fmt.Errorf("aws cli script fallback failed: %w", err)
	}
	if debug {
		_, _ = fmt.Fprintln(os.Stderr, "[jumphost] script fallback succeeded")
	}
	return nil
}

func isTTYError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "/dev/tty") || strings.Contains(msg,
		"could not open a new tty") || strings.Contains(msg, "device not configured")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...<truncated>"
}

func finalizeErr(err error) error {
	if err == nil {
		return nil
	}
	if isTTYError(err) {
		return fmt.Errorf("%v\nRemediation suggestions:\n 1. Set JUMPHOST_FORCE_PTY=1 (export JUMPHOST_FORCE_PTY=1) and retry\n 2. Enable debug for diagnostics: export JUMPHOST_DEBUG=1\n 3. Run the binary from an external terminal instead of the IDE debugger\n 4. Ensure the AWS Session Manager plugin is installed and up to date (aws --version & session-manager-plugin --version)\n 5. macOS only: Grant the IDE Full Disk Access & Terminal permissions and restart\n 6. If still failing, force 'script' fallback: ensure /usr/bin/script exists (macOS default) and try again",
			err)
	}
	return err
}

// preflightAWSDeps ensures aws CLI and session-manager-plugin exist and prints their versions to stderr.
func preflightAWSDeps(ctx context.Context) error {
	if _, err := exec.LookPath("aws"); err != nil {
		return fmt.Errorf("aws cli not found in PATH: %w", err)
	}
	// capture and print aws --version
	awsVerCmd := exec.CommandContext(ctx, "aws", "--version")
	var awsOut bytes.Buffer
	awsVerCmd.Stdout = &awsOut
	awsVerCmd.Stderr = &awsOut // aws historically writes to stderr; capture both
	_ = awsVerCmd.Run()        // non-critical; ignore error to avoid blocking usage if version fails
	if s := strings.TrimSpace(awsOut.String()); s != "" {
		_, _ = fmt.Fprintf(os.Stderr, "[jumphost] aws version: %s\n", s)
	}
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		return fmt.Errorf("session-manager-plugin not found in PATH: %w\nInstall instructions: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
			err)
	}
	plugVerCmd := exec.CommandContext(ctx, "session-manager-plugin", "--version")
	var plugOut bytes.Buffer
	plugVerCmd.Stdout = &plugOut
	plugVerCmd.Stderr = &plugOut
	_ = plugVerCmd.Run() // ignore errors, best-effort
	if s := strings.TrimSpace(plugOut.String()); s != "" {
		_, _ = fmt.Fprintf(os.Stderr, "[jumphost] session-manager-plugin version: %s\n", s)
	}
	return nil
}
