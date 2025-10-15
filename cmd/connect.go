package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
	"github.com/Oleexo/jumphost-cli/internal/hosts"
	"github.com/Oleexo/jumphost-cli/internal/session"
	"github.com/Oleexo/jumphost-cli/internal/ui/basic"
	"github.com/Oleexo/jumphost-cli/internal/ui/connectflow"
	"github.com/Oleexo/jumphost-cli/internal/ui/jumphostselect"
)

// dependency injection hooks (overridden in tests)
var (
	loadConfigFunc          = awsclient.LoadDefaultConfig
	getIdentityFunc         = awsclient.GetIdentity
	validateCredentialsFunc = awsclient.ValidateCredentials
	newEC2Func              = awsclient.NewEC2
	newRDSFunc              = awsclient.NewRDS
	runConnectFlowFunc      = connectflow.Run
	runJumpSelectFunc       = jumphostselect.Run
	startPortForwarding     = session.StartPortForwarding
	listJumpHostInstancesFn = func(e *awsclient.EC2, ctx context.Context, tag string) ([]string, error) {
		return e.ListJumpHostInstances(ctx, tag)
	}
	runConnectFlowBasicFunc = func(
		ctx context.Context,
		rds *awsclient.RDS,
		in io.Reader,
		out io.Writer) (connectflow.Result, error) {
		return basic.RunConnectFlowBasic(ctx, rds, in, out)
	}
	runJumpSelectBasicFunc = func(ctx context.Context, ids []string, in io.Reader, out io.Writer) (
		string,
		bool,
		error) {
		return basic.RunJumpHostSelectBasic(ctx, ids, in, out)
	}
)

func newConnectCmd() *cobra.Command {
	var (
		jumphostTag      string
		jumphostInstance string
		noHosts          bool
		endpointArg      string
		localPortArg     int
		interactive      bool
		region           string
		profile          string
		dryRun           bool
		basicMode        bool
		noPTY            bool
		jsonMode         bool
		verbose          bool
	)

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to an RDS instance through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Env/company overrides
			if os.Getenv("JUMPHOST_BASIC") == "1" {
				basicMode = true
			}
			// Auto-enable basic mode ONLY if not explicitly interactive and stdio not TTY
			if !basicMode && !interactive {
				stdinTTY := isatty.IsTerminal(os.Stdin.Fd())
				stdoutTTY := isatty.IsTerminal(os.Stdout.Fd())
				if !stdinTTY || !stdoutTTY {
					basicMode = true
				}
			}
			if jsonMode && (interactive || endpointArg == "") { // enforce basic for JSON interactive flows
				basicMode = true
			}

			emit := func(ev any) {
				if !jsonMode {
					return
				}
				b, _ := json.Marshal(ev)
				fmt.Println(string(b))
			}
			vlog := func(format string, a ...any) {
				if !verbose {
					return
				}
				if jsonMode {
					emit(map[string]any{"type": "log", "level": "info", "message": fmt.Sprintf(format, a...)})
				} else {
					_, _ = fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", a...)
				}
			}

			// If no profile flag provided, inherit from AWS_PROFILE env var (if set).
			if profile == "" {
				if envProf := os.Getenv("AWS_PROFILE"); envProf != "" {
					profile = envProf
				}
			}

			cfg, err := loadConfigFunc(ctx, region, profile)
			if err != nil {
				if jsonMode {
					emit(map[string]any{"type": "error", "message": err.Error()})
					return nil
				}
				return fmt.Errorf("load aws config: %w", err)
			}
			emit(map[string]any{"type": "config_loaded", "region": cfg.Region, "profile": profile})
			vlog("Loaded AWS config region=%s profile=%s", cfg.Region, profile)

			// Pre-flight credential validation
			authMethod := awsclient.DetectAuthMethod(profile)
			vlog("Detected authentication method: %s", authMethod)

			if err := validateCredentialsFunc(ctx, cfg, profile); err != nil {
				if authErr, isAuth := awsclient.IsAuthError(err); isAuth {
					if jsonMode {
						emit(map[string]any{
							"type":        "auth_error",
							"message":     authErr.Message,
							"hint":        authErr.Hint,
							"is_sso":      authErr.IsSSOErr,
							"profile":     authErr.Profile,
							"auth_method": authMethod,
						})
						return nil
					}
					// Pretty print for non-JSON mode
					_, _ = fmt.Fprintf(os.Stderr, "\n❌ %s\n", authErr.Message)
					if authErr.Hint != "" {
						_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", authErr.Hint)
					}
					_, _ = fmt.Fprintf(os.Stderr, "\nAuthentication method: %s\n", authMethod)
					return fmt.Errorf("credential validation failed")
				}
				// Non-auth error
				if jsonMode {
					emit(map[string]any{"type": "error", "message": err.Error()})
					return nil
				}
				return fmt.Errorf("credential validation: %w", err)
			}
			vlog("Credentials validated successfully")

			identity, idErr := getIdentityFunc(ctx, cfg)
			if idErr != nil {
				vlog("Failed to resolve identity: %v", idErr)
				if jsonMode {
					emit(map[string]any{"type": "identity_error", "message": idErr.Error()})
				}
			} else {
				if jsonMode {
					emit(map[string]any{
						"type": "identity", "account": identity.Account, "user": identity.UserID, "arn": identity.Arn,
					})
				} else if identity.String() != "" {
					fmt.Println(identity.String())
				}
				vlog("Resolved identity account=%s user=%s", identity.Account, identity.UserID)
			}

			ec2c := newEC2Func(cfg)
			rdsc := newRDSFunc(cfg)

			var endpoint string
			var remotePort int
			var localPort int

			if interactive || endpointArg == "" { // interactive flows
				if basicMode {
					res, err := runConnectFlowBasicFunc(ctx, rdsc, os.Stdin, os.Stdout)
					if err != nil {
						if jsonMode {
							emit(map[string]any{"type": "error", "message": err.Error()})
							return nil
						}
						return err
					}
					if res.Cancelled {
						emit(map[string]any{"type": "cancelled"})
						if !jsonMode {
							fmt.Println("Cancelled")
						}
						return nil
					}
					endpoint, remotePort, localPort = res.Endpoint, res.RemotePort, res.LocalPort
					noHosts = noHosts || !res.ApplyHosts
					emit(map[string]any{
						"type": "selection", "endpoint": endpoint, "remotePort": remotePort, "localPort": localPort,
						"applyHosts": !noHosts,
					})
				} else {
					m := connectflow.NewModel(ctx, rdsc, identity)
					res, err := runConnectFlowFunc(m)
					if err != nil {
						if jsonMode {
							emit(map[string]any{"type": "error", "message": err.Error()})
							return nil
						}
						return err
					}
					if res.Cancelled {
						emit(map[string]any{"type": "cancelled"})
						if !jsonMode {
							fmt.Println("Cancelled")
						}
						return nil
					}
					endpoint, remotePort, localPort = res.Endpoint, res.RemotePort, res.LocalPort
					noHosts = noHosts || !res.ApplyHosts
					emit(map[string]any{
						"type": "selection", "endpoint": endpoint, "remotePort": remotePort, "localPort": localPort,
						"applyHosts": !noHosts,
					})
				}
			} else {
				var parseErr error
				endpoint, remotePort, parseErr = parseEndpoint(endpointArg)
				if parseErr != nil {
					if jsonMode {
						emit(map[string]any{"type": "error", "message": parseErr.Error()})
						return nil
					}
					return parseErr
				}
				if localPortArg != 0 {
					localPort = localPortArg
				} else {
					localPort = remotePort
				}
				emit(map[string]any{
					"type": "selection", "endpoint": endpoint, "remotePort": remotePort, "localPort": localPort,
					"applyHosts": !noHosts,
				})
			}

			if endpoint == "" || remotePort == 0 {
				err := errors.New("missing endpoint or port")
				if jsonMode {
					emit(map[string]any{"type": "error", "message": err.Error()})
					return nil
				}
				return err
			}
			if localPort == 0 {
				localPort = remotePort
			}

			// Select jumphost instance
			var instanceID string
			if jumphostInstance != "" {
				instanceID = jumphostInstance
			} else {
				ids, err := listJumpHostInstancesFn(ec2c, ctx, jumphostTag)
				if err != nil {
					if jsonMode {
						emit(map[string]any{"type": "error", "message": err.Error()})
						return nil
					}
					return fmt.Errorf("list jumphost instances: %w", err)
				}
				if len(ids) == 0 {
					err = errors.New("no jumphost instance found")
					if jsonMode {
						emit(map[string]any{"type": "error", "message": err.Error()})
						return nil
					}
					return err
				}
				if len(ids) == 1 || (!interactive && endpointArg != "") {
					instanceID = ids[0]
				} else {
					if basicMode {
						id, cancelled, err := runJumpSelectBasicFunc(ctx, ids, os.Stdin, os.Stdout)
						if err != nil {
							if jsonMode {
								emit(map[string]any{"type": "error", "message": err.Error()})
								return nil
							}
							return err
						}
						if cancelled {
							emit(map[string]any{"type": "cancelled"})
							if !jsonMode {
								fmt.Println("Cancelled")
							}
							return nil
						}
						instanceID = id
					} else {
						jm := jumphostselect.NewModel(ctx, ids)
						jr, err := runJumpSelectFunc(jm)
						if err != nil {
							if jsonMode {
								emit(map[string]any{"type": "error", "message": err.Error()})
								return nil
							}
							return err
						}
						if jr.Cancelled {
							emit(map[string]any{"type": "cancelled"})
							if !jsonMode {
								fmt.Println("Cancelled")
							}
							return nil
						}
						instanceID = jr.InstanceID
					}
				}
			}
			emit(map[string]any{"type": "instance_selected", "instanceID": instanceID})
			vlog("Selected jumphost instance: %s", instanceID)

			if dryRun {
				emit(map[string]any{
					"type": "dry_run", "instanceID": instanceID, "endpoint": endpoint, "remotePort": remotePort,
					"localPort": localPort, "region": cfg.Region, "profile": profile, "applyHosts": !noHosts,
				})
				if !jsonMode {
					fmt.Printf("[dry-run] Would establish port forward: %s %s:%d -> localhost:%d (region=%s profile=%s)\n",
						instanceID, endpoint, remotePort, localPort, cfg.Region, profile)
					if !noHosts {
						fmt.Printf("[dry-run] Would add /etc/hosts entry: 127.0.0.1 %s\n", endpoint)
					}
				}
				vlog("Dry-run complete")
				return nil
			}

			var hm *hosts.Manager
			if !noHosts {
				hm = hosts.NewDefaultManager()
				applied, herr := hm.Apply(endpoint)
				if herr != nil {
					if jsonMode {
						emit(map[string]any{"type": "hosts_error", "message": herr.Error()})
					} else {
						_, _ = fmt.Fprintf(os.Stderr, "warn: failed to modify hosts file: %v\n", herr)
					}
				} else if applied {
					emit(map[string]any{"type": "hosts_applied", "endpoint": endpoint})
					if !jsonMode {
						defer func() { _ = hm.Restore() }()
						fmt.Printf("Added temporary hosts entry for %s -> 127.0.0.1 (will restore on exit)\n", endpoint)
					} else {
						defer func() { _ = hm.Restore() }()
					}
				}
			}
			// After hosts logic ends but before session start output
			if noHosts {
				vlog("Hosts modification skipped")
			}

			region = cfg.Region
			if profile != "" {
				_ = os.Setenv("AWS_PROFILE", profile)
			}
			if noPTY {
				_ = os.Setenv("JUMPHOST_DISABLE_PTY", "1")
			}
			emit(map[string]any{
				"type": "session_start", "instanceID": instanceID, "endpoint": endpoint, "remotePort": remotePort,
				"localPort": localPort, "region": region,
			})
			vlog("Starting session instance=%s endpoint=%s remotePort=%d localPort=%d region=%s noPTY=%v", instanceID,
				endpoint, remotePort, localPort, region, os.Getenv("JUMPHOST_DISABLE_PTY") == "1")
			if !jsonMode {
				fmt.Printf("Starting port forwarding session %s %s:%d -> localhost:%d in region %s...\n", instanceID,
					endpoint, remotePort, localPort, region)
			}
			err = startPortForwarding(ctx, instanceID, endpoint, remotePort, localPort, region)
			if err != nil {
				// Handle graceful cancellation (CTRL+C)
				if errors.Is(err, context.Canceled) {
					emit(map[string]any{"type": "session_cancelled"})
					vlog("Session cancelled by user")
					if !jsonMode {
						fmt.Println("\nSession cancelled by user.")
					}
					return nil
				}
				if jsonMode {
					emit(map[string]any{"type": "error", "message": err.Error()})
					return nil
				}
				return err
			}
			emit(map[string]any{"type": "session_end"})
			vlog("Session ended cleanly")
			if !jsonMode {
				fmt.Println("Session ended. Have a safe day !")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&jumphostTag, "jumphost-tag", "jumphost", "EC2 tag value for Usage to pick jumphost instance")
	cmd.Flags().StringVar(&jumphostInstance, "jumphost-instance", "",
		"Explicit jumphost instance id to use (skip selection)")
	cmd.Flags().BoolVar(&noHosts, "no-hosts", false, "Do not modify /etc/hosts to map endpoint to localhost")
	cmd.Flags().StringVarP(&endpointArg, "endpoint", "e", "", "Endpoint in the form host:port (non-interactive)")
	cmd.Flags().IntVarP(&localPortArg, "local-port", "l", 0, "Local port to bind (defaults to remote port)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Force interactive TUI even if endpoint provided")
	cmd.Flags().StringVar(&region, "region", "", "AWS region override (defaults to AWS configuration)")
	cmd.Flags().StringVar(&profile, "profile", "", "AWS shared config profile (supports SSO profiles)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"Show actions without executing (skip hosts modification and session)")
	cmd.Flags().BoolVar(&basicMode, "basic", false, "Force basic (non-TUI) prompts; auto-used when no TTY available")
	cmd.Flags().BoolVar(&noPTY, "no-pty", false, "Disable PTY/script fallbacks for aws session (direct exec only)")
	cmd.Flags().BoolVar(&jsonMode, "json", false,
		"Emit machine-readable JSON lines to stdout (implies --basic for interactive flows)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (JSON: emits log events)")
	return cmd
}

func parseEndpoint(in string) (string, int, error) {
	if !strings.Contains(in, ":") {
		return "", 0, fmt.Errorf("endpoint must be host:port")
	}
	host, portStr, _ := strings.Cut(in, ":")
	if host == "" {
		return "", 0, fmt.Errorf("empty host")
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p <= 0 || p > 65535 {
		return "", 0, fmt.Errorf("invalid port")
	}
	// Allow arbitrary hostnames and IP addresses
	return host, p, nil
}
