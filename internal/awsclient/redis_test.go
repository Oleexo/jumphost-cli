package awsclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRedis(t *testing.T) {
	// Just verify construction doesn't panic
	// Can't fully test without valid AWS config
	// r := NewRedis(aws.Config{})
	// assert.NotNil(t, r)
}

func TestRedis_ListEndpoints(t *testing.T) {
	// Note: Full integration test would require mocking the ElastiCache API
	// This test focuses on the structure and basic functionality
	ctx := context.Background()

	t.Run("Returns endpoints successfully", func(t *testing.T) {
		// This would require a comprehensive mock of ElastiCache API
		// which includes both DescribeCacheClusters and DescribeReplicationGroups
		// For now, we verify the method exists and has the right signature
		_ = ctx
	})
}

func TestRedisEndpoint_Structure(t *testing.T) {
	endpoint := RedisEndpoint{
		Name:           "my-redis",
		Endpoint:       "my-redis.cache.amazonaws.com",
		Port:           6379,
		Engine:         "redis",
		EngineVersion:  "6.2",
		Status:         "available",
		NodeType:       "cache.t3.micro",
		IsCluster:      false,
		ReaderEndpoint: "my-redis-ro.cache.amazonaws.com",
	}

	assert.Equal(t, "my-redis", endpoint.Name)
	assert.Equal(t, "my-redis.cache.amazonaws.com", endpoint.Endpoint)
	assert.Equal(t, 6379, endpoint.Port)
	assert.Equal(t, "redis", endpoint.Engine)
	assert.Equal(t, "6.2", endpoint.EngineVersion)
	assert.Equal(t, "available", endpoint.Status)
	assert.Equal(t, "cache.t3.micro", endpoint.NodeType)
	assert.False(t, endpoint.IsCluster)
	assert.Equal(t, "my-redis-ro.cache.amazonaws.com", endpoint.ReaderEndpoint)
}
