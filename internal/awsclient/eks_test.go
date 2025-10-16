package awsclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEKSClient for testing
type MockEKSClient struct {
	cluster *types.Cluster
	err     error
}

func (m *MockEKSClient) DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &eks.DescribeClusterOutput{
		Cluster: m.cluster,
	}, nil
}

func TestNewEKS(t *testing.T) {
	// Just verify construction doesn't panic
	// Can't fully test without valid AWS config
	// e := NewEKS(aws.Config{})
	// assert.NotNil(t, e)
}

func TestEKS_GetCluster(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns cluster successfully", func(t *testing.T) {
		endpoint := "https://EXAMPLE123.gr7.us-east-1.eks.amazonaws.com"
		mockClient := &MockEKSClient{
			cluster: &types.Cluster{
				Name:     stringPtr("my-cluster"),
				Endpoint: &endpoint,
			},
		}

		e := &EKS{client: mockClient}
		cluster, err := e.GetCluster(ctx, "my-cluster")

		require.NoError(t, err)
		assert.Equal(t, "my-cluster", cluster.Name)
		assert.Equal(t, endpoint, cluster.Endpoint)
	})

	t.Run("Returns error when API fails", func(t *testing.T) {
		mockClient := &MockEKSClient{
			err: fmt.Errorf("cluster not found"),
		}

		e := &EKS{client: mockClient}
		cluster, err := e.GetCluster(ctx, "nonexistent-cluster")

		assert.Error(t, err)
		assert.Equal(t, "", cluster.Name)
		assert.Equal(t, "", cluster.Endpoint)
	})

	t.Run("Returns empty cluster when response has no cluster", func(t *testing.T) {
		mockClient := &MockEKSClient{
			cluster: nil,
		}

		e := &EKS{client: mockClient}
		cluster, err := e.GetCluster(ctx, "my-cluster")

		require.NoError(t, err)
		assert.Equal(t, "", cluster.Name)
		assert.Equal(t, "", cluster.Endpoint)
	})

	t.Run("Returns empty cluster when endpoint is nil", func(t *testing.T) {
		mockClient := &MockEKSClient{
			cluster: &types.Cluster{
				Name:     stringPtr("my-cluster"),
				Endpoint: nil,
			},
		}

		e := &EKS{client: mockClient}
		cluster, err := e.GetCluster(ctx, "my-cluster")

		require.NoError(t, err)
		assert.Equal(t, "", cluster.Name)
		assert.Equal(t, "", cluster.Endpoint)
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
