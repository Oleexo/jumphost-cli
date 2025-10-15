package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	version = "0.2.0"
	commit  = "dev"
	date    = "unknown"
)

// NewRootCmd constructs the root command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "jumphost",
		Short: "Interactive AWS RDS port forwarding via SSM jumphost",
		Long:  "jumphost provides an interactive TUI to select an RDS instance and create a port forwarding session through an EC2 instance tagged Usage=jumphost.",
		RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}

	cmd.AddCommand(newConnectCmd())
	cmd.AddCommand(&cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("jumphost %s (commit %s, built %s)\n", version, commit, date)
		},
	})
	return cmd
}

// Execute runs the root command with graceful cancellation on SIGINT/SIGTERM.
func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := NewRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
