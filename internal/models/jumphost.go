package models

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type JumphostInstance struct {
	InstanceID       string
	Name             string
	PrivateIPAddress string
	State            string
}

func (i JumphostInstance) Start(ctx context.Context, params ConnectionParams) error {
	ssmParams := map[string][]string{
		"host":            {params.Endpoint},
		"portNumber":      {fmt.Sprintf("%d", params.RemotePort)},
		"localPortNumber": {fmt.Sprintf("%d", params.LocalPort)},
	}

	paramsJSON, err := json.Marshal(ssmParams)
	if err != nil {
		return fmt.Errorf("failed to marshal SSM parameters: %w", err)
	}

	// Build AWS SSM command
	args := []string{
		"ssm", "start-session",
		"--target", i.InstanceID,
		"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters", string(paramsJSON),
	}

	if params.Region != "" {
		args = append(args, "--region", params.Region)
	}

	if params.Profile != "" {
		args = append(args, "--profile", params.Profile)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSM session: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-sigChan:
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		return fmt.Errorf("session interrupted by user")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("SSM session failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ctx.Err()
	}
}

type ConnectionParams struct {
	Endpoint    string
	RemotePort  int
	LocalPort   int
	InstanceID  string
	ApplyHosts  bool
	Region      string
	Profile     string
	ServiceName string
	ClusterName string
}

func (cp ConnectionParams) DisplayName() string {
	if cp.ClusterName != "" {
		return cp.ClusterName
	}
	return cp.Endpoint
}

func (cp ConnectionParams) Hostname() string {
	return cp.Endpoint
}

func (cp ConnectionParams) Port() int {
	return cp.RemotePort
}
