package cmd

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Oleexo/jumphost-cli/internal/models"
	"github.com/Oleexo/jumphost-cli/internal/ui"
)

// Mock identity for testing
type mockIdentity struct {
	account string
	arn     string
	userID  string
}

func (m mockIdentity) Account() string { return m.account }
func (m mockIdentity) Arn() string     { return m.arn }
func (m mockIdentity) UserID() string  { return m.userID }

// Mock display for testing
type mockDisplay struct {
	selectedService         models.Service
	selectedTarget          models.ConnectionParams
	selectedJumphost        models.JumphostInstance
	selectedRegion          string
	selectedLocalPort       int
	shouldQuit              bool
	printedErrors           []string
	identity                mockIdentity
	loadingFunc             func() (any, error)
	selectJumphostFunc      func() ([]models.JumphostInstance, error)
	selectTargetFunc        func() ([]models.ConnectionParams, error)
	startJumphostCalled     bool
	startJumphostInstance   models.JumphostInstance
	startJumphostConnection models.ConnectionParams
}

func (m *mockDisplay) SetIdentity(identity ui.Identity) {
	// Accept any identity type
}

func (m *mockDisplay) SelectService(services []models.Service) (models.Service, bool) {
	if m.shouldQuit {
		return models.Service{}, true
	}
	if m.selectedService.Key != models.ServiceTypeNone {
		return m.selectedService, false
	}
	if len(services) > 0 {
		return services[0], false
	}
	return models.Service{}, true
}

func (m *mockDisplay) SelectJumphostInstance(loader func() ([]models.JumphostInstance, error)) (
	models.JumphostInstance,
	bool,
	error) {
	if m.selectJumphostFunc != nil {
		loader = m.selectJumphostFunc
	}
	instances, err := loader()
	if err != nil {
		return models.JumphostInstance{}, false, err
	}
	if m.shouldQuit {
		return models.JumphostInstance{}, true, nil
	}
	if m.selectedJumphost.InstanceID != "" {
		return m.selectedJumphost, false, nil
	}
	if len(instances) > 0 {
		return instances[0], false, nil
	}
	return models.JumphostInstance{}, true, nil
}

func (m *mockDisplay) SelectTarget(
	service models.Service,
	loader func() ([]models.ConnectionParams, error)) (models.ConnectionParams, bool) {
	if m.selectTargetFunc != nil {
		loader = m.selectTargetFunc
	}
	targets, err := loader()
	if err != nil || m.shouldQuit {
		return models.ConnectionParams{}, true
	}
	if m.selectedTarget.Endpoint != "" {
		return m.selectedTarget, false
	}
	if len(targets) > 0 {
		return targets[0], false
	}
	return models.ConnectionParams{}, true
}

func (m *mockDisplay) SelectRegion(regions []models.Region) models.Region {
	if m.selectedRegion != "" {
		return models.Region{Code: m.selectedRegion, Name: m.selectedRegion}
	}
	if len(regions) > 0 {
		return regions[0]
	}
	return models.Region{}
}

func (m *mockDisplay) SelectLocalPort(remotePort int) int {
	if m.selectedLocalPort > 0 {
		return m.selectedLocalPort
	}
	return remotePort
}

func (m *mockDisplay) Loading(message string, loader func() (any, error)) (any, error) {
	if m.loadingFunc != nil {
		return m.loadingFunc()
	}
	return loader()
}

func (m *mockDisplay) StartJumphost(instance models.JumphostInstance, params models.ConnectionParams) error {
	m.startJumphostCalled = true
	m.startJumphostInstance = instance
	m.startJumphostConnection = params
	return nil
}

func (m *mockDisplay) PrintErrorf(format string, args ...interface{}) {
	m.printedErrors = append(m.printedErrors, format)
}

func (m *mockDisplay) Print(message string) {
}

func (m *mockDisplay) PrintError(message string, err error) {
	m.printedErrors = append(m.printedErrors, message)
}

// Mock EC2 client
type mockEC2 struct {
	instances    []models.JumphostInstance
	instanceByID models.JumphostInstance
	listError    error
	getError     error
}

func (m *mockEC2) ListJumpHostInstances(ctx context.Context, tag string) ([]models.JumphostInstance, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.instances, nil
}

func (m *mockEC2) GetJumpHostInstance(ctx context.Context, instanceID string) (models.JumphostInstance, error) {
	if m.getError != nil {
		return models.JumphostInstance{}, m.getError
	}
	return m.instanceByID, nil
}

func TestGetService(t *testing.T) {
	tests := []struct {
		name       string
		serviceKey models.ServiceType
		shouldQuit bool
		expectQuit bool
		expectKey  models.ServiceType
	}{
		{
			name:       "Returns RDS when requested",
			serviceKey: models.ServiceTypeRDS,
			shouldQuit: false,
			expectQuit: false,
			expectKey:  models.ServiceTypeRDS,
		},
		{
			name:       "Returns OpenSearch when requested",
			serviceKey: models.ServiceTypeOpenSearch,
			shouldQuit: false,
			expectQuit: false,
			expectKey:  models.ServiceTypeOpenSearch,
		},
		{
			name:       "Returns first service when none specified",
			serviceKey: models.ServiceTypeNone,
			shouldQuit: false,
			expectQuit: false,
			expectKey:  models.ServiceTypeRDS,
		},
		{
			name:       "Returns quit when display quits",
			serviceKey: models.ServiceTypeNone,
			shouldQuit: true,
			expectQuit: true,
			expectKey:  models.ServiceTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			display := &mockDisplay{shouldQuit: tt.shouldQuit}
			service, quit := getService(display, tt.serviceKey)
			assert.Equal(t, tt.expectQuit, quit)
			if !quit {
				assert.Equal(t, tt.expectKey, service.Key)
			}
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		expectedHost string
		expectedPort int
		expectError  bool
	}{
		{
			name:         "Empty endpoint",
			endpoint:     "",
			expectedHost: "",
			expectedPort: 0,
			expectError:  false,
		},
		{
			name:         "Hostname only",
			endpoint:     "db.example.com",
			expectedHost: "db.example.com",
			expectedPort: 0,
			expectError:  false,
		},
		{
			name:         "Hostname with port",
			endpoint:     "db.example.com:5432",
			expectedHost: "db.example.com",
			expectedPort: 5432,
			expectError:  false,
		},
		{
			name:         "IP with port",
			endpoint:     "10.0.1.5:3306",
			expectedHost: "10.0.1.5",
			expectedPort: 3306,
			expectError:  false,
		},
		{
			name:         "Hostname with colon but no port",
			endpoint:     "db.example.com:",
			expectedHost: "db.example.com",
			expectedPort: 0,
			expectError:  false,
		},
		{
			name:         "Invalid port",
			endpoint:     "db.example.com:abc",
			expectedHost: "db.example.com:abc",
			expectedPort: 0,
			expectError:  false, // Function returns the whole string as hostname when it can't parse port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := parseEndpoint(tt.endpoint)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHost, host)
				assert.Equal(t, tt.expectedPort, port)
			}
		})
	}
}

func TestFilterConnectionByEndpoint(t *testing.T) {
	connections := []models.ConnectionParams{
		{Endpoint: "db1.example.com", RemotePort: 5432, LocalPort: 5432},
		{Endpoint: "db2.example.com", RemotePort: 3306, LocalPort: 3306},
		{Endpoint: "db3.example.com", RemotePort: 5432, LocalPort: 5432},
	}

	tests := []struct {
		name          string
		endpoint      string
		expectedCount int
		expectedPort  int
	}{
		{
			name:          "Empty endpoint returns all",
			endpoint:      "",
			expectedCount: 3,
		},
		{
			name:          "Match by hostname",
			endpoint:      "db1.example.com",
			expectedCount: 1,
		},
		{
			name:          "Match with port override",
			endpoint:      "db1.example.com:9999",
			expectedCount: 1,
			expectedPort:  9999,
		},
		{
			name:          "No match returns all",
			endpoint:      "nonexistent.example.com",
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterConnectionByEndpoint(connections, tt.endpoint)
			assert.Equal(t, tt.expectedCount, len(result))
			if tt.expectedPort > 0 && len(result) == 1 {
				assert.Equal(t, tt.expectedPort, result[0].RemotePort)
				assert.Equal(t, tt.expectedPort, result[0].LocalPort)
			}
		})
	}
}

func TestGetJumphostInstance_WithInstanceID(t *testing.T) {
	ctx := context.Background()
	expectedInstance := models.JumphostInstance{
		InstanceID: "i-12345",
		Name:       "jumphost-1",
	}

	ec2 := &mockEC2{
		instanceByID: expectedInstance,
	}

	display := &mockDisplay{}

	instance, quit := getJumphostInstance(ctx, ec2, display, "", "i-12345")

	assert.False(t, quit)
	assert.Equal(t, expectedInstance.InstanceID, instance.InstanceID)
}

func TestGetJumphostInstance_SelectFromList(t *testing.T) {
	ctx := context.Background()
	instances := []models.JumphostInstance{
		{InstanceID: "i-11111", Name: "jumphost-1"},
		{InstanceID: "i-22222", Name: "jumphost-2"},
	}

	ec2 := &mockEC2{
		instances: instances,
	}

	display := &mockDisplay{}

	instance, quit := getJumphostInstance(ctx, ec2, display, "jumphost", "")

	assert.False(t, quit)
	assert.Equal(t, instances[0].InstanceID, instance.InstanceID)
}

func TestGetJumphostInstance_UserQuits(t *testing.T) {
	ctx := context.Background()
	ec2 := &mockEC2{
		instances: []models.JumphostInstance{
			{InstanceID: "i-11111", Name: "jumphost-1"},
		},
	}

	display := &mockDisplay{shouldQuit: true}

	_, quit := getJumphostInstance(ctx, ec2, display, "jumphost", "")

	assert.True(t, quit)
}

func TestGetConnectionParamsFromRDS(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	loader := getConnectionParamsFromRDS(ctx, cfg, "")
	require.NotNil(t, loader)

	// Test that the function is created correctly
	// Full integration test would require mocking RDS client
}

func TestGetConnectionParamsFromOpenSearch(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	loader := getConnectionParamsFromOpenSearch(ctx, cfg, "")
	require.NotNil(t, loader)
}

func TestGetConnectionParamsFromRedshift(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	loader := getConnectionParamsFromRedshift(ctx, cfg, "")
	require.NotNil(t, loader)
}

func TestGetConnectionParamsFromRedis(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	loader := getConnectionParamsFromRedis(ctx, cfg, "")
	require.NotNil(t, loader)
}

func TestGetConnectionParamsFromDocumentDB(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	loader := getConnectionParamsFromDocumentDB(ctx, cfg, "")
	require.NotNil(t, loader)
}

func TestGetConnectionParamsFromEKS(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	t.Run("Returns error when cluster name is missing", func(t *testing.T) {
		loader := getConnectionParamsFromEKS(ctx, cfg, "")
		require.NotNil(t, loader)

		_, err := loader()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cluster name is required")
	})

	t.Run("Creates loader with cluster name", func(t *testing.T) {
		loader := getConnectionParamsFromEKS(ctx, cfg, "my-cluster")
		require.NotNil(t, loader)
	})
}

func TestGetConnectionParams(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	tests := []struct {
		name       string
		serviceKey models.ServiceType
		endpoint   string
		localPort  int
		shouldQuit bool
		expectQuit bool
	}{
		{
			name:       "RDS service with endpoint",
			serviceKey: models.ServiceTypeRDS,
			endpoint:   "db.example.com",
			localPort:  5432,
			shouldQuit: false,
			expectQuit: false, // Won't quit because we provide working data
		},
		{
			name:       "OpenSearch service",
			serviceKey: models.ServiceTypeOpenSearch,
			endpoint:   "",
			localPort:  0,
			shouldQuit: false,
			expectQuit: false, // Won't quit because we provide working data
		},
		{
			name:       "User quits selection",
			serviceKey: models.ServiceTypeRDS,
			shouldQuit: true,
			expectQuit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			display := &mockDisplay{
				shouldQuit: tt.shouldQuit,
				selectedTarget: models.ConnectionParams{
					Endpoint:   "test.example.com",
					RemotePort: 5432,
					LocalPort:  5432,
				},
				selectTargetFunc: func() ([]models.ConnectionParams, error) {
					if tt.shouldQuit {
						return nil, nil
					}
					return []models.ConnectionParams{
						{Endpoint: "test.example.com", RemotePort: 5432, LocalPort: 5432},
					}, nil
				},
			}

			service := models.Service{Key: tt.serviceKey}
			_, quit := getConnectionParams(ctx, cfg, display, service, tt.endpoint, tt.localPort)

			assert.Equal(t, tt.expectQuit, quit)
		})
	}
}

func TestGetConnectionParams_UnsupportedService(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}
	display := &mockDisplay{}

	service := models.Service{Key: models.ServiceTypeNone}
	_, quit := getConnectionParams(ctx, cfg, display, service, "", 0)

	assert.True(t, quit)
	assert.NotEmpty(t, display.printedErrors)
}

func TestGetConnectionParams_WithLocalPortOverride(t *testing.T) {
	ctx := context.Background()
	cfg := aws.Config{}

	display := &mockDisplay{
		selectedTarget: models.ConnectionParams{
			Endpoint:   "test.example.com",
			RemotePort: 5432,
			LocalPort:  5432,
		},
		selectTargetFunc: func() ([]models.ConnectionParams, error) {
			return []models.ConnectionParams{
				{Endpoint: "test.example.com", RemotePort: 5432, LocalPort: 5432},
			}, nil
		},
	}

	service := models.Service{Key: models.ServiceTypeRDS}
	params, quit := getConnectionParams(ctx, cfg, display, service, "test.example.com", 9999)

	assert.False(t, quit)
	assert.Equal(t, 9999, params.LocalPort)
}
