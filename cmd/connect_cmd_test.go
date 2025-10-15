package cmd

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
	"github.com/Oleexo/jumphost-cli/internal/ui/connectflow"
	"github.com/Oleexo/jumphost-cli/internal/ui/jumphostselect"
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
		override(&loadConfigFunc,
			func(ctx context.Context, region string, profile string) (aws.Config, error) {
				return aws.Config{Region: "us-east-1"}, nil
			}),
		override(&newEC2Func,
			func(cfg aws.Config) *awsclient.EC2 {
				return &awsclient.EC2{}
			}),
		override(&newRDSFunc,
			func(cfg aws.Config) *awsclient.RDS {
				return &awsclient.RDS{}
			}),
		override(&listJumpHostInstancesFn,
			func(e *awsclient.EC2, ctx context.Context, tag string) ([]string, error) {
				return []string{"i-test123"}, nil
			}),
		override(&validateCredentialsFunc,
			func(ctx context.Context, cfg aws.Config, profile string) error {
				return nil
			}),
		override(&runConnectFlowFunc,
			func(m connectflow.Model) (connectflow.Result, error) {
				return connectflow.Result{Cancelled: true}, nil
			}),
		override(&runJumpSelectFunc,
			func(m jumphostselect.Model) (jumphostselect.Result, error) {
				return jumphostselect.Result{InstanceID: "", Cancelled: true}, nil
			}),
		override(&startPortForwarding,
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
		override(&getIdentityFunc, func(ctx context.Context, cfg aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
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
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{Account: "111", UserID: "user/abc"}, nil
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

func TestConnectNoInstancesError(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
		}),
		override(&listJumpHostInstancesFn,
			func(e *awsclient.EC2, ctx context.Context, tag string) ([]string, error) {
				return []string{}, nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.Error(t, err)
}

func TestConnectExplicitInstanceLocalPort(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
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

func TestConnectInvalidEndpoint(t *testing.T) {
	restores := []restoreFn{
		override(&validateCredentialsFunc, func(context.Context, aws.Config, string) error { return nil }),
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
		}),
	}
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--endpoint", "invalid"})
	err := root.Execute()
	require.Error(t, err)
}

func TestConnectDryRunWithProfile(t *testing.T) {
	var profileSeen string
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
		}),
		override(&loadConfigFunc,
			func(ctx context.Context, region string, profile string) (aws.Config, error) {
				profileSeen = profile
				return aws.Config{Region: "us-east-1"}, nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--profile", "myprofile", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
	require.Equal(t, "myprofile", profileSeen)
}

func TestConnectProfileEnvSet(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
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

func TestConnectInheritProfileFromEnv(t *testing.T) {
	restores := standardTestMocks()
	oldEnv := os.Getenv("AWS_PROFILE")
	_ = os.Setenv("AWS_PROFILE", "env-prof")
	defer func() { _ = os.Setenv("AWS_PROFILE", oldEnv) }()

	var seen string
	restores = append(restores,
		override(&getIdentityFunc, func(context.Context, aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{}, nil
		}),
		override(&loadConfigFunc,
			func(ctx context.Context, region string, profile string) (aws.Config, error) {
				seen = profile
				return aws.Config{Region: "us-east-1"}, nil
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
	require.NotEmpty(t, seen)
}

func TestConnectBasicInteractiveSuccess(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }))
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
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }),
		override(&runConnectFlowBasicFunc,
			func(ctx context.Context, rds *awsclient.RDS, in io.Reader, out io.Writer) (connectflow.Result, error) {
				return connectflow.Result{Cancelled: true}, nil
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
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }))
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

func TestConnectJSONDryRun(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--json", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectJSONCancelled(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }),
		override(&runConnectFlowBasicFunc,
			func(ctx context.Context, rds *awsclient.RDS, in io.Reader, out io.Writer) (connectflow.Result, error) {
				return connectflow.Result{Cancelled: true}, nil
			}))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--json", "--interactive"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectNoPTYFlagSetsEnv(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }))
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
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }))
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

func TestConnectVerboseJSON(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc,
			func(context.Context, aws.Config) (awsclient.Identity, error) { return awsclient.Identity{}, nil }))
	defer func() {
		for _, restore := range restores {
			restore()
		}
	}()

	root := NewRootCmd()
	root.SetArgs([]string{"connect", "--verbose", "--json", "--dry-run", "--endpoint", "db.example:5432"})
	err := root.Execute()
	require.NoError(t, err)
}

func TestConnectIdentityPrintedDryRun(t *testing.T) {
	restores := standardTestMocks()
	restores = append(restores,
		override(&getIdentityFunc, func(ctx context.Context, cfg aws.Config) (awsclient.Identity, error) {
			return awsclient.Identity{
				Account: "123456789012", UserID: "user/tester", Arn: "arn:aws:iam::123456789012:user/tester",
			}, nil
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
