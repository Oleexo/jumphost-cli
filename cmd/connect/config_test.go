package connect

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "jumphost", cfg.JumphostTag)
	assert.Equal(t, "", cfg.JumphostInstance)
	assert.False(t, cfg.NoHosts)
	assert.Equal(t, "", cfg.EndpointArg)
	assert.Equal(t, 0, cfg.LocalPortArg)
	assert.False(t, cfg.Interactive)
	assert.Equal(t, "", cfg.Region)
	assert.Equal(t, "", cfg.Profile)
	assert.False(t, cfg.DryRun)
	assert.False(t, cfg.BasicMode)
	assert.False(t, cfg.NoPTY)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "", cfg.EKSCluster)
}

func TestDetectBasicMode_EnvVariable(t *testing.T) {
	oldVal := os.Getenv("JUMPHOST_BASIC")
	defer func() {
		if oldVal == "" {
			_ = os.Unsetenv("JUMPHOST_BASIC")
		} else {
			_ = os.Setenv("JUMPHOST_BASIC", oldVal)
		}
	}()

	_ = os.Setenv("JUMPHOST_BASIC", "1")

	cfg := NewConfig()
	cfg.detectBasicMode()

	assert.True(t, cfg.BasicMode)
}

func TestDetectBasicMode_NoEnvVariable(t *testing.T) {
	oldVal := os.Getenv("JUMPHOST_BASIC")
	defer func() {
		if oldVal == "" {
			_ = os.Unsetenv("JUMPHOST_BASIC")
		} else {
			_ = os.Setenv("JUMPHOST_BASIC", oldVal)
		}
	}()

	_ = os.Unsetenv("JUMPHOST_BASIC")

	cfg := NewConfig()
	cfg.detectBasicMode()

	// Result depends on whether we're in a TTY or not
	// Just verify it doesn't panic
	assert.NotNil(t, cfg)
}

func TestDetectBasicMode_InteractiveModeIgnoresTTY(t *testing.T) {
	oldVal := os.Getenv("JUMPHOST_BASIC")
	defer func() {
		if oldVal == "" {
			_ = os.Unsetenv("JUMPHOST_BASIC")
		} else {
			_ = os.Setenv("JUMPHOST_BASIC", oldVal)
		}
	}()

	_ = os.Unsetenv("JUMPHOST_BASIC")

	cfg := NewConfig()
	cfg.Interactive = true
	originalBasicMode := cfg.BasicMode

	cfg.detectBasicMode()

	// Interactive mode should prevent TTY detection from changing BasicMode
	assert.Equal(t, originalBasicMode, cfg.BasicMode)
}

func TestBindFlags(t *testing.T) {
	cfg := NewConfig()
	cmd := &cobra.Command{
		Use: "test",
	}

	cfg.BindFlags(cmd)

	// Verify all flags are defined
	assert.NotNil(t, cmd.Flags().Lookup("jumphost-tag"))
	assert.NotNil(t, cmd.Flags().Lookup("jumphost-instance"))
	assert.NotNil(t, cmd.Flags().Lookup("no-hosts"))
	assert.NotNil(t, cmd.Flags().Lookup("endpoint"))
	assert.NotNil(t, cmd.Flags().Lookup("local-port"))
	assert.NotNil(t, cmd.Flags().Lookup("interactive"))
	assert.NotNil(t, cmd.Flags().Lookup("region"))
	assert.NotNil(t, cmd.Flags().Lookup("profile"))
	assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
	assert.NotNil(t, cmd.Flags().Lookup("basic"))
	assert.NotNil(t, cmd.Flags().Lookup("no-pty"))
	assert.NotNil(t, cmd.Flags().Lookup("verbose"))

	// Verify shorthand flags
	assert.NotNil(t, cmd.Flags().ShorthandLookup("e"))
	assert.NotNil(t, cmd.Flags().ShorthandLookup("l"))
	assert.NotNil(t, cmd.Flags().ShorthandLookup("i"))
	assert.NotNil(t, cmd.Flags().ShorthandLookup("v"))
}

func TestBindFlags_ValueBinding(t *testing.T) {
	cfg := NewConfig()
	cmd := &cobra.Command{
		Use: "test",
	}

	cfg.BindFlags(cmd)

	// Set flag values
	_ = cmd.Flags().Set("jumphost-tag", "my-jumphost")
	_ = cmd.Flags().Set("jumphost-instance", "i-12345")
	_ = cmd.Flags().Set("endpoint", "db.example.com:5432")
	_ = cmd.Flags().Set("local-port", "9999")
	_ = cmd.Flags().Set("region", "us-west-2")
	_ = cmd.Flags().Set("profile", "my-profile")
	_ = cmd.Flags().Set("no-hosts", "true")
	_ = cmd.Flags().Set("interactive", "true")
	_ = cmd.Flags().Set("dry-run", "true")
	_ = cmd.Flags().Set("basic", "true")
	_ = cmd.Flags().Set("no-pty", "true")
	_ = cmd.Flags().Set("verbose", "true")

	// Verify config values are updated
	assert.Equal(t, "my-jumphost", cfg.JumphostTag)
	assert.Equal(t, "i-12345", cfg.JumphostInstance)
	assert.Equal(t, "db.example.com:5432", cfg.EndpointArg)
	assert.Equal(t, 9999, cfg.LocalPortArg)
	assert.Equal(t, "us-west-2", cfg.Region)
	assert.Equal(t, "my-profile", cfg.Profile)
	assert.True(t, cfg.NoHosts)
	assert.True(t, cfg.Interactive)
	assert.True(t, cfg.DryRun)
	assert.True(t, cfg.BasicMode)
	assert.True(t, cfg.NoPTY)
	assert.True(t, cfg.Verbose)
}

func TestBindEKSFlags(t *testing.T) {
	cfg := NewConfig()
	cmd := &cobra.Command{
		Use: "test",
	}

	cfg.BindEKSFlags(cmd)

	// Verify EKS-specific flag
	clusterFlag := cmd.Flags().Lookup("cluster")
	assert.NotNil(t, clusterFlag)

	// Verify all base flags are also present
	assert.NotNil(t, cmd.Flags().Lookup("jumphost-tag"))
	assert.NotNil(t, cmd.Flags().Lookup("endpoint"))

	// Verify endpoint flag usage is overridden
	endpointFlag := cmd.Flags().Lookup("endpoint")
	assert.Contains(t, endpointFlag.Usage, "overrides --cluster")

	// Verify interactive flag usage is overridden
	interactiveFlag := cmd.Flags().Lookup("interactive")
	assert.Contains(t, interactiveFlag.Usage, "cluster name")
}

func TestBindEKSFlags_ClusterBinding(t *testing.T) {
	cfg := NewConfig()
	cmd := &cobra.Command{
		Use: "test",
	}

	cfg.BindEKSFlags(cmd)

	// Set cluster flag
	_ = cmd.Flags().Set("cluster", "my-eks-cluster")

	assert.Equal(t, "my-eks-cluster", cfg.EKSCluster)
}

func TestConfig_AllFieldsSettable(t *testing.T) {
	cfg := &Config{
		JumphostTag:      "custom-tag",
		JumphostInstance: "i-custom",
		NoHosts:          true,
		EndpointArg:      "custom.endpoint:1234",
		LocalPortArg:     8080,
		Interactive:      true,
		Region:           "eu-west-1",
		Profile:          "custom-profile",
		DryRun:           true,
		BasicMode:        true,
		NoPTY:            true,
		Verbose:          true,
		EKSCluster:       "my-cluster",
	}

	// Verify all fields are set
	assert.Equal(t, "custom-tag", cfg.JumphostTag)
	assert.Equal(t, "i-custom", cfg.JumphostInstance)
	assert.True(t, cfg.NoHosts)
	assert.Equal(t, "custom.endpoint:1234", cfg.EndpointArg)
	assert.Equal(t, 8080, cfg.LocalPortArg)
	assert.True(t, cfg.Interactive)
	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, "custom-profile", cfg.Profile)
	assert.True(t, cfg.DryRun)
	assert.True(t, cfg.BasicMode)
	assert.True(t, cfg.NoPTY)
	assert.True(t, cfg.Verbose)
	assert.Equal(t, "my-cluster", cfg.EKSCluster)
}
