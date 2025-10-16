package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"

	"github.com/Oleexo/jumphost-cli/cmd/connect"
	"github.com/Oleexo/jumphost-cli/internal/awsclient"
	"github.com/Oleexo/jumphost-cli/internal/models"
)

// restoreFn is a function that restores a previous state
type restoreFn func()

// helper to override and restore a function variable
func override[T any](orig *T, repl T) restoreFn {
	prev := *orig
	*orig = repl
	return func() { *orig = prev }
}

// standardTestMocks returns common test mocks for AWS operations
func standardTestMocks() []restoreFn {
	return []restoreFn{
		override(&connect.LoadConfigFunc,
			func(ctx context.Context, region string, profile string) (aws.Config, error) {
				return aws.Config{Region: "us-east-1"}, nil
			}),
		override(&connect.NewEC2Func,
			func(cfg aws.Config) awsclient.EC2 {
				return awsclient.NewEC2(cfg)
			}),
		override(&connect.NewRDSFunc,
			func(cfg aws.Config) *awsclient.RDS {
				return &awsclient.RDS{}
			}),
		override(&connect.ListJumpHostInstancesFn,
			func(e awsclient.EC2, ctx context.Context, tag string) ([]models.JumphostInstance, error) {
				return []models.JumphostInstance{{InstanceID: "i-test123", Name: "test"}}, nil
			}),
		override(&connect.ValidateCredentialsFunc,
			func(ctx context.Context, cfg aws.Config, profile string) error {
				return nil
			}),
		override(&connect.StartPortForwarding,
			func(ctx context.Context, instanceID, host string, remotePort, localPort int, region string) error {
				return nil
			}),
	}
}

func TestConnectDryRunSingleInstance(t *testing.T) {
	restores := standardTestMocks()
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectInteractiveCancelled(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc, func(ctx context.Context, cfg aws.Config) (awsclient.Identity, error) {
			return awsclient.NewIdentityForTest("", "", ""), nil
		}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--interactive"})
	err := root.Execute()
	// Cancelled should not be an error
	require.NoError(t, err)
}

func TestConnectInteractiveSuccessMultipleInstances(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.NewIdentityForTest("111", "", "user/abc"), nil
		}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--interactive"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectExplicitInstanceLocalPort(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.NewIdentityForTest("", "", ""), nil
		}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{
		"connect", "--jumphost-instance", "i-abc123", "--endpoint", "db.example:5432", "--local-port", "9999",
		"--dry-run",
	})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectBasicInteractiveSuccess(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--basic", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectBasicCancelled(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--basic", "--interactive"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectBasicEnvAuto(t *testing.T) {
	old := os.Getenv("JUMPHOST_BASIC")
	_ = os.Setenv("JUMPHOST_BASIC", "1")
	defer func() { _ = os.Setenv("JUMPHOST_BASIC", old) }()

	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectNoPTYFlagSetsEnv(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--no-pty", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectVerbosePlain(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--verbose", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectIdentityPrintedDryRun(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc, func(ctx context.Context, cfg aws.Config) (awsclient.Identity, error) {
			return awsclient.NewIdentityForTest("123456789012", "arn:aws:iam::123456789012:user/tester",
				"user/tester"), nil
		}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--verbose", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectOpenSearchDryRun(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "opensearch", "--dry-run", "--endpoint", "os.example:443"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectRedshiftDryRun(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "redshift", "--dry-run", "--endpoint", "cluster.example:5439"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectEKSDryRun(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&connect.GetIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) {
				return awsclient.NewIdentityForTest("", "", ""), nil
			}),
		override(&connect.NewEKSFunc, func(cfg aws.Config) *awsclient.EKS { return &awsclient.EKS{} }),
		override(&connect.GetEKSClusterFn,
			func(e *awsclient.EKS, ctx context.Context, name string) (awsclient.Cluster, error) {
				return awsclient.Cluster{Name: name, Endpoint: "demo.eks.local"}, nil
			}),
	)
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "eks", "--cluster", "demo", "--dry-run"})
	err := root.Execute()
	require.NoError(t, err)
}
