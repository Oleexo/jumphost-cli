package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_Name(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected string
	}{
		{
			name: "RDS service",
			service: Service{
				Key:         ServiceTypeRDS,
				Title:       "RDS",
				Description: "Relational Database Service",
			},
			expected: "RDS",
		},
		{
			name: "OpenSearch service",
			service: Service{
				Key:         ServiceTypeOpenSearch,
				Title:       "OpenSearch",
				Description: "OpenSearch domain",
			},
			expected: "OpenSearch",
		},
		{
			name: "Empty service",
			service: Service{
				Key:         ServiceTypeNone,
				Title:       "",
				Description: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.service.Name())
		})
	}
}

func TestServiceType_String(t *testing.T) {
	tests := []struct {
		name        string
		serviceType ServiceType
		expected    string
	}{
		{
			name:        "None service type",
			serviceType: ServiceTypeNone,
			expected:    "",
		},
		{
			name:        "RDS service type",
			serviceType: ServiceTypeRDS,
			expected:    "rds",
		},
		{
			name:        "OpenSearch service type",
			serviceType: ServiceTypeOpenSearch,
			expected:    "opensearch",
		},
		{
			name:        "Redshift service type",
			serviceType: ServiceTypeRedshift,
			expected:    "redshift",
		},
		{
			name:        "Redis service type",
			serviceType: ServiceTypeRedis,
			expected:    "redis",
		},
		{
			name:        "DocumentDB service type",
			serviceType: ServiceTypeDocumentDB,
			expected:    "documentdb",
		},
		{
			name:        "EKS service type",
			serviceType: ServiceTypeEKS,
			expected:    "eks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.serviceType.String())
		})
	}
}

func TestService_AllFields(t *testing.T) {
	service := Service{
		Key:         ServiceTypeRDS,
		Title:       "RDS",
		Description: "Relational Database Service",
	}

	assert.Equal(t, ServiceTypeRDS, service.Key)
	assert.Equal(t, "RDS", service.Title)
	assert.Equal(t, "Relational Database Service", service.Description)
	assert.Equal(t, "RDS", service.Name())
}

func TestServiceTypeConstants(t *testing.T) {
	// Verify all service type constants are defined correctly
	assert.Equal(t, ServiceType(""), ServiceTypeNone)
	assert.Equal(t, ServiceType("rds"), ServiceTypeRDS)
	assert.Equal(t, ServiceType("opensearch"), ServiceTypeOpenSearch)
	assert.Equal(t, ServiceType("redshift"), ServiceTypeRedshift)
	assert.Equal(t, ServiceType("redis"), ServiceTypeRedis)
	assert.Equal(t, ServiceType("documentdb"), ServiceTypeDocumentDB)
	assert.Equal(t, ServiceType("eks"), ServiceTypeEKS)
}
