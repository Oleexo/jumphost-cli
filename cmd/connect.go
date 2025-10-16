package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/Oleexo/jumphost-cli/cmd/connect"
	"github.com/Oleexo/jumphost-cli/internal/awsclient"
	"github.com/Oleexo/jumphost-cli/internal/models"
	"github.com/Oleexo/jumphost-cli/internal/ui"
	"github.com/Oleexo/jumphost-cli/internal/ui/basic"
	"github.com/Oleexo/jumphost-cli/internal/ui/tui"
)

func getDisplay(cfg *connect.Config) ui.Display {
	useTUI := os.Getenv("JUMPHOST_BASIC") != "1" && !cfg.BasicMode &&
		isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())

	var display ui.Display
	if useTUI {
		display = tui.NewDisplay()
	} else {
		display = basic.NewDisplay()
	}
	return display
}

func getService(display ui.Display, serviceKey models.ServiceType) (models.Service, bool) {
	services := []models.Service{
		{Key: models.ServiceTypeRDS, Title: "RDS", Description: "Relational Database Service"},
		{Key: models.ServiceTypeOpenSearch, Title: "OpenSearch", Description: "OpenSearch domain"},
		{Key: models.ServiceTypeRedshift, Title: "Redshift", Description: "Redshift cluster"},
		{Key: models.ServiceTypeRedis, Title: "Redis (ElastiCache)", Description: "ElastiCache Redis endpoint"},
		{Key: models.ServiceTypeDocumentDB, Title: "DocumentDB", Description: "Amazon DocumentDB cluster"},
		{Key: models.ServiceTypeEKS, Title: "EKS (cluster API)", Description: "EKS cluster API endpoint"},
	}

	if serviceKey != models.ServiceTypeNone {
		for _, s := range services {
			if s.Key == serviceKey {
				return s, false
			}
		}
	}
	return display.SelectService(services)
}

func getAwsConfig(
	ctx context.Context,
	display ui.Display,
	region string,
	profile string) (aws.Config, error) {
	if profile == "" {
		if envProf := os.Getenv("AWS_PROFILE"); envProf != "" {
			profile = envProf
		}
	}
	if region == "" {
		if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
			region = envRegion
		}
		if region == "" {
			region = "us-east-1"
		}
	}

	cfg, err := awsclient.LoadDefaultConfig(ctx, region, profile)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws config: %w", err)
	}

	awsclient.DetectAuthMethod(profile)

	if err := awsclient.ValidateCredentials(ctx, cfg, profile); err != nil {
		if authErr, isAuth := awsclient.IsAuthError(err); isAuth {
			return aws.Config{}, fmt.Errorf("credential validation failed: %s", authErr.Message)
		}
		return aws.Config{}, fmt.Errorf("credential validation: %w", err)
	}

	identity, idErr := awsclient.GetIdentity(ctx, cfg)
	if idErr != nil {
		return aws.Config{}, fmt.Errorf("failed to resolve identity: %w", idErr)
	} else {
		display.SetIdentity(identity)
	}

	return cfg, nil
}

// parseEndpoint parses an endpoint string in the format "host:port" or just "host".
// Returns the host, port (0 if not specified), and any error.
func parseEndpoint(endpoint string) (host string, port int, err error) {
	if endpoint == "" {
		return "", 0, nil
	}

	// Check if endpoint contains a port
	if idx := len(endpoint) - 1; idx > 0 {
		for i := idx; i >= 0; i-- {
			if endpoint[i] == ':' {
				host = endpoint[:i]
				portStr := endpoint[i+1:]
				if portStr != "" {
					parsedPort := 0
					_, scanErr := fmt.Sscanf(portStr, "%d", &parsedPort)
					if scanErr != nil {
						return "", 0, fmt.Errorf("invalid port in endpoint: %s", portStr)
					}
					return host, parsedPort, nil
				}
				return host, 0, nil
			}
			// Stop if we hit a non-numeric character that's not ':'
			if endpoint[i] < '0' || endpoint[i] > '9' {
				break
			}
		}
	}

	// No port found, return the whole string as host
	return endpoint, 0, nil
}

// filterConnectionByEndpoint filters connections to match the provided endpoint.
// If endpoint is provided, returns a single-element slice with matching connection,
// or empty slice if no match. If endpoint is empty, returns all connections.
func filterConnectionByEndpoint(connections []models.ConnectionParams, endpoint string) []models.ConnectionParams {
	if endpoint == "" {
		return connections
	}

	host, port, err := parseEndpoint(endpoint)
	if err != nil {
		return connections // Return all if parsing fails
	}

	for _, conn := range connections {
		if conn.Endpoint == host || conn.Endpoint == endpoint {
			if port > 0 {
				conn.RemotePort = port
				conn.LocalPort = port
			}
			return []models.ConnectionParams{conn}
		}
	}

	return connections
}

func getJumphostInstance(
	ctx context.Context,
	ec2 awsclient.EC2,
	display ui.Display,
	jumphostInstanceTag string,
	jumphostInstanceID string) (models.JumphostInstance, bool) {
	if jumphostInstanceID == "" {
		instance, quit, err := display.SelectJumphostInstance(func() ([]models.JumphostInstance, error) {
			instances, err := ec2.ListJumpHostInstances(ctx, jumphostInstanceTag)
			if err != nil {
				return nil, err
			}
			return instances, nil
		})
		if err != nil {
			display.PrintErrorf("", err)
			return models.JumphostInstance{}, true
		}
		if quit {
			return models.JumphostInstance{}, true
		}

		return instance, false
	} else {
		i, err := display.Loading("Getting jumphost instance...", func() (any, error) {
			return ec2.GetJumpHostInstance(ctx, jumphostInstanceID)
		})
		if err != nil {
			display.PrintErrorf("", err)
			return models.JumphostInstance{}, true
		}
		return i.(models.JumphostInstance), false
	}
}

func getConnectionParamsFromRDS(
	ctx context.Context, awsCfg aws.Config,
	endpoint string) func() ([]models.ConnectionParams, error) {
	return func() ([]models.ConnectionParams, error) {
		rdsClient := awsclient.NewRDS(awsCfg)
		endpoints, err := rdsClient.ListEndpoints(ctx)
		if err != nil {
			return nil, err
		}
		connections := make([]models.ConnectionParams, len(endpoints))
		for i, e := range endpoints {
			connections[i] = models.ConnectionParams{
				Endpoint:   e.Address,
				RemotePort: e.Port,
				LocalPort:  e.Port,
			}
		}
		return filterConnectionByEndpoint(connections, endpoint), nil
	}
}

func getConnectionParamsFromOpenSearch(
	ctx context.Context, awsCfg aws.Config,
	endpoint string) func() ([]models.ConnectionParams, error) {
	return func() ([]models.ConnectionParams, error) {
		client := awsclient.NewOpenSearch(awsCfg)
		domains, err := client.ListDomains(ctx)
		if err != nil {
			return nil, err
		}
		connections := make([]models.ConnectionParams, len(domains))
		for i, d := range domains {
			connections[i] = models.ConnectionParams{
				Endpoint:   d.Endpoint,
				RemotePort: d.Port,
				LocalPort:  d.Port,
			}
		}
		return filterConnectionByEndpoint(connections, endpoint), nil
	}
}

func getConnectionParamsFromRedshift(
	ctx context.Context, awsCfg aws.Config,
	endpoint string) func() ([]models.ConnectionParams, error) {
	return func() ([]models.ConnectionParams, error) {
		client := awsclient.NewRedshift(awsCfg)
		clusters, err := client.ListClusters(ctx)
		if err != nil {
			return nil, err
		}
		connections := make([]models.ConnectionParams, len(clusters))
		for i, c := range clusters {
			connections[i] = models.ConnectionParams{
				Endpoint:   c.Endpoint,
				RemotePort: c.Port,
				LocalPort:  c.Port,
			}
		}
		return filterConnectionByEndpoint(connections, endpoint), nil
	}
}

func getConnectionParamsFromRedis(
	ctx context.Context, awsCfg aws.Config,
	endpoint string) func() ([]models.ConnectionParams, error) {
	return func() ([]models.ConnectionParams, error) {
		client := awsclient.NewRedis(awsCfg)
		endpoints, err := client.ListEndpoints(ctx)
		if err != nil {
			return nil, err
		}
		connections := make([]models.ConnectionParams, len(endpoints))
		for i, e := range endpoints {
			connections[i] = models.ConnectionParams{
				Endpoint:   e.Endpoint,
				RemotePort: e.Port,
				LocalPort:  e.Port,
			}
		}
		return filterConnectionByEndpoint(connections, endpoint), nil
	}
}

func getConnectionParamsFromDocumentDB(
	ctx context.Context, awsCfg aws.Config,
	endpoint string) func() ([]models.ConnectionParams, error) {
	return func() ([]models.ConnectionParams, error) {
		client := awsclient.NewDocumentDB(awsCfg)
		clusters, err := client.ListClusters(ctx)
		if err != nil {
			return nil, err
		}
		connections := make([]models.ConnectionParams, len(clusters))
		for i, c := range clusters {
			connections[i] = models.ConnectionParams{
				Endpoint:   c.Endpoint,
				RemotePort: c.Port,
				LocalPort:  c.Port,
			}
		}
		return filterConnectionByEndpoint(connections, endpoint), nil
	}
}

func getConnectionParamsFromEKS(
	ctx context.Context, awsCfg aws.Config,
	endpoint string) func() ([]models.ConnectionParams, error) {
	return func() ([]models.ConnectionParams, error) {
		// For EKS, we need the cluster name from the endpoint parameter
		if endpoint == "" {
			return nil, fmt.Errorf("cluster name is required for EKS connections")
		}
		client := awsclient.NewEKS(awsCfg)
		cluster, err := client.GetCluster(ctx, endpoint)
		if err != nil {
			return nil, err
		}
		// EKS API endpoints use HTTPS on port 443
		return []models.ConnectionParams{
			{
				Endpoint:   cluster.Endpoint,
				RemotePort: 443,
				LocalPort:  443,
			},
		}, nil
	}
}

func getConnectionParams(
	ctx context.Context,
	awsCfg aws.Config,
	display ui.Display,
	service models.Service,
	endpoint string,
	localPort int) (models.ConnectionParams, bool) {
	var loader func() ([]models.ConnectionParams, error)
	switch service.Key {
	case models.ServiceTypeRDS:
		loader = getConnectionParamsFromRDS(ctx, awsCfg, endpoint)
	case models.ServiceTypeOpenSearch:
		loader = getConnectionParamsFromOpenSearch(ctx, awsCfg, endpoint)
	case models.ServiceTypeRedshift:
		loader = getConnectionParamsFromRedshift(ctx, awsCfg, endpoint)
	case models.ServiceTypeRedis:
		loader = getConnectionParamsFromRedis(ctx, awsCfg, endpoint)
	case models.ServiceTypeDocumentDB:
		loader = getConnectionParamsFromDocumentDB(ctx, awsCfg, endpoint)
	case models.ServiceTypeEKS:
		loader = getConnectionParamsFromEKS(ctx, awsCfg, endpoint)
	default:
		display.PrintErrorf("Unsupported service: %s", service.Key)
		return models.ConnectionParams{}, true
	}
	params, quit := display.SelectTarget(service, loader)
	if quit {
		return models.ConnectionParams{}, true
	}

	if localPort > 0 {
		params.LocalPort = localPort
	} else {
		params.LocalPort = display.SelectLocalPort(params.RemotePort)
	}

	return params, false
}

func run(
	ctx context.Context,
	cfg *connect.Config,
	serviceKey models.ServiceType) error {
	display := getDisplay(cfg)

	awsCfg, err := getAwsConfig(ctx, display, cfg.Region, cfg.Profile)
	if err != nil {
		display.PrintError("Cannot get aws config", err)
		return nil
	}

	service, quit := getService(display, serviceKey)
	if quit {
		// todo add quit message
		return nil
	}

	jumphostInstance, quit := getJumphostInstance(ctx,
		awsclient.NewEC2(awsCfg),
		display,
		cfg.JumphostTag,
		cfg.JumphostInstance)
	if quit {
		// todo add quit message
		return nil
	}

	info, quit := getConnectionParams(ctx,
		awsCfg,
		display,
		service,
		cfg.EndpointArg,
		cfg.LocalPortArg)
	if quit {
		// todo add quit message
		return nil
	}

	return display.StartJumphost(jumphostInstance, info)
}

func createSubcommands(cfg *connect.Config) []*cobra.Command {
	rdsCmd := &cobra.Command{
		Use:   "rds",
		Short: "Connect to an RDS instance through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeRDS)
		},
	}
	cfg.BindFlags(rdsCmd)

	opensearchCmd := &cobra.Command{
		Use:   "opensearch",
		Short: "Connect to an OpenSearch domain through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeOpenSearch)
		},
	}
	cfg.BindFlags(opensearchCmd)

	redshiftCmd := &cobra.Command{
		Use:   "redshift",
		Short: "Connect to a Redshift cluster through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeRedshift)
		},
	}
	cfg.BindFlags(redshiftCmd)

	redisCmd := &cobra.Command{
		Use:   "redis",
		Short: "Connect to a Redis (ElastiCache) endpoint through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeRedis)
		},
	}
	cfg.BindFlags(redisCmd)

	docdbCmd := &cobra.Command{
		Use:   "docdb",
		Short: "Connect to an Amazon DocumentDB cluster through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeDocumentDB)
		},
	}
	cfg.BindFlags(docdbCmd)

	eksCmd := &cobra.Command{
		Use:   "eks",
		Short: "Connect to an EKS cluster API endpoint through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeEKS)
		},
	}
	cfg.BindEKSFlags(eksCmd)

	return []*cobra.Command{rdsCmd, opensearchCmd, redshiftCmd, redisCmd, docdbCmd, eksCmd}
}

func newConnectCmd() *cobra.Command {
	cfg := connect.NewConfig()

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect to an AWS service through a jumphost (SSM port forwarding)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), cfg, models.ServiceTypeNone)
		},
	}

	cfg.BindFlags(cmd)
	cmd.AddCommand(createSubcommands(cfg)...)

	return cmd
}
