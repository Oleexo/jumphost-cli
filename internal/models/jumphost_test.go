package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJumphostInstance_Structure(t *testing.T) {
	instance := JumphostInstance{
		InstanceID:       "i-1234567890abcdef0",
		Name:             "jumphost-1",
		PrivateIPAddress: "10.0.1.5",
		State:            "running",
	}

	assert.Equal(t, "i-1234567890abcdef0", instance.InstanceID)
	assert.Equal(t, "jumphost-1", instance.Name)
	assert.Equal(t, "10.0.1.5", instance.PrivateIPAddress)
	assert.Equal(t, "running", instance.State)
}

func TestConnectionParams_DisplayName(t *testing.T) {
	tests := []struct {
		name     string
		params   ConnectionParams
		expected string
	}{
		{
			name: "Returns cluster name when present",
			params: ConnectionParams{
				Endpoint:    "db.example.com",
				ClusterName: "my-cluster",
			},
			expected: "my-cluster",
		},
		{
			name: "Returns endpoint when cluster name is empty",
			params: ConnectionParams{
				Endpoint:    "db.example.com",
				ClusterName: "",
			},
			expected: "db.example.com",
		},
		{
			name: "Returns endpoint when only endpoint is set",
			params: ConnectionParams{
				Endpoint: "opensearch.example.com",
			},
			expected: "opensearch.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.params.DisplayName())
		})
	}
}

func TestConnectionParams_Hostname(t *testing.T) {
	params := ConnectionParams{
		Endpoint:   "db.example.com",
		RemotePort: 5432,
		LocalPort:  5432,
	}

	assert.Equal(t, "db.example.com", params.Hostname())
}

func TestConnectionParams_Port(t *testing.T) {
	params := ConnectionParams{
		Endpoint:   "db.example.com",
		RemotePort: 5432,
		LocalPort:  9999,
	}

	assert.Equal(t, 5432, params.Port())
}

func TestConnectionParams_AllFields(t *testing.T) {
	params := ConnectionParams{
		Endpoint:    "db.example.com",
		RemotePort:  5432,
		LocalPort:   9999,
		InstanceID:  "i-12345",
		ApplyHosts:  true,
		Region:      "us-east-1",
		Profile:     "default",
		ServiceName: "rds",
		ClusterName: "my-cluster",
	}

	assert.Equal(t, "db.example.com", params.Endpoint)
	assert.Equal(t, 5432, params.RemotePort)
	assert.Equal(t, 9999, params.LocalPort)
	assert.Equal(t, "i-12345", params.InstanceID)
	assert.True(t, params.ApplyHosts)
	assert.Equal(t, "us-east-1", params.Region)
	assert.Equal(t, "default", params.Profile)
	assert.Equal(t, "rds", params.ServiceName)
	assert.Equal(t, "my-cluster", params.ClusterName)

	// Test methods
	assert.Equal(t, "my-cluster", params.DisplayName())
	assert.Equal(t, "db.example.com", params.Hostname())
	assert.Equal(t, 5432, params.Port())
}

func TestConnectionParams_DefaultValues(t *testing.T) {
	params := ConnectionParams{}

	assert.Equal(t, "", params.Endpoint)
	assert.Equal(t, 0, params.RemotePort)
	assert.Equal(t, 0, params.LocalPort)
	assert.Equal(t, "", params.InstanceID)
	assert.False(t, params.ApplyHosts)
	assert.Equal(t, "", params.Region)
	assert.Equal(t, "", params.Profile)
	assert.Equal(t, "", params.ServiceName)
	assert.Equal(t, "", params.ClusterName)
}

func TestJumphostInstance_DefaultValues(t *testing.T) {
	instance := JumphostInstance{}

	assert.Equal(t, "", instance.InstanceID)
	assert.Equal(t, "", instance.Name)
	assert.Equal(t, "", instance.PrivateIPAddress)
	assert.Equal(t, "", instance.State)
}
