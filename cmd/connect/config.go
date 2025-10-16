package connect

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// Config holds all configuration for the connect command
type Config struct {
	JumphostTag      string
	JumphostInstance string
	NoHosts          bool
	EndpointArg      string
	LocalPortArg     int
	Interactive      bool
	Region           string
	Profile          string
	DryRun           bool
	BasicMode        bool
	NoPTY            bool
	Verbose          bool
	EKSCluster       string
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		JumphostTag: "jumphost",
	}
}

// detectBasicMode auto-enables basic mode when appropriate
func (c *Config) detectBasicMode() {
	if os.Getenv("JUMPHOST_BASIC") == "1" {
		c.BasicMode = true
	}
	if !c.BasicMode && !c.Interactive {
		stdinTTY := isatty.IsTerminal(os.Stdin.Fd())
		stdoutTTY := isatty.IsTerminal(os.Stdout.Fd())
		if !stdinTTY || !stdoutTTY {
			c.BasicMode = true
		}
	}
}

// BindFlags binds configuration flags to a cobra command
func (c *Config) BindFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.JumphostTag, "jumphost-tag", "jumphost",
		"EC2 tag value for Usage to pick jumphost instance")
	cmd.Flags().StringVar(&c.JumphostInstance, "jumphost-instance", "",
		"Explicit jumphost instance id to use (skip selection)")
	cmd.Flags().BoolVar(&c.NoHosts, "no-hosts", false, "Do not modify /etc/hosts to map endpoint to localhost")
	cmd.Flags().StringVarP(&c.EndpointArg, "endpoint", "e", "", "Endpoint in the form host:port (non-interactive)")
	cmd.Flags().IntVarP(&c.LocalPortArg, "local-port", "l", 0, "Local port to bind (defaults to remote port)")
	cmd.Flags().BoolVarP(&c.Interactive, "interactive", "i", false, "Force interactive TUI even if endpoint provided")
	cmd.Flags().StringVar(&c.Region, "region", "", "AWS region override (defaults to AWS configuration)")
	cmd.Flags().StringVar(&c.Profile, "profile", "", "AWS shared config profile (supports SSO profiles)")
	cmd.Flags().BoolVar(&c.DryRun, "dry-run", false,
		"Show actions without executing (skip hosts modification and session)")
	cmd.Flags().BoolVar(&c.BasicMode, "basic", false, "Force basic (non-TUI) prompts; auto-used when no TTY available")
	cmd.Flags().BoolVar(&c.NoPTY, "no-pty", false, "Disable PTY/script fallbacks for aws session (direct exec only)")
	cmd.Flags().BoolVarP(&c.Verbose, "verbose", "v", false, "Enable verbose logging")
}

// BindEKSFlags binds EKS-specific configuration flags to a cobra command
func (c *Config) BindEKSFlags(cmd *cobra.Command) {
	c.BindFlags(cmd)
	cmd.Flags().StringVar(&c.EKSCluster, "cluster", "", "EKS cluster name to resolve (alternatively use --endpoint)")
	// Override endpoint flag description for EKS
	cmd.Flags().Lookup("endpoint").Usage = "Endpoint in the form host:port (non-interactive; overrides --cluster)"
	cmd.Flags().Lookup("interactive").Usage = "Interactive prompt for cluster name if --cluster not provided"
}
