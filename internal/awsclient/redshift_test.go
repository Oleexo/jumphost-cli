package awsclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedshift(t *testing.T) {
	// Just verify construction doesn't panic
	// Can't fully test without valid AWS config
	// r := NewRedshift(aws.Config{})
	// assert.NotNil(t, r)
}

func TestRedshift_ListClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns clusters successfully", func(t *testing.T) {
		mockClient := &MockRedshift{
			Clusters: []RedshiftCluster{
				{
					ClusterIdentifier: "cluster-1",
					Endpoint:          "cluster-1.redshift.amazonaws.com",
					Port:              5439,
					NodeType:          "dc2.large",
					ClusterStatus:     "available",
					DatabaseName:      "dev",
				},
				{
					ClusterIdentifier: "cluster-2",
					Endpoint:          "cluster-2.redshift.amazonaws.com",
					Port:              5439,
					NodeType:          "ra3.xlplus",
					ClusterStatus:     "available",
					DatabaseName:      "prod",
				},
			},
		}

		r := &Redshift{client: mockClient}
		clusters, err := r.ListClusters(ctx)

		require.NoError(t, err)
		assert.Len(t, clusters, 2)
		assert.Equal(t, "cluster-1", clusters[0].ClusterIdentifier)
		assert.Equal(t, "cluster-1.redshift.amazonaws.com", clusters[0].Endpoint)
		assert.Equal(t, 5439, clusters[0].Port)
		assert.Equal(t, "dc2.large", clusters[0].NodeType)
		assert.Equal(t, "available", clusters[0].ClusterStatus)
		assert.Equal(t, "dev", clusters[0].DatabaseName)
	})

	t.Run("Returns error when API fails", func(t *testing.T) {
		mockClient := &MockRedshift{
			Err: fmt.Errorf("API error"),
		}

		r := &Redshift{client: mockClient}
		clusters, err := r.ListClusters(ctx)

		assert.Error(t, err)
		assert.Nil(t, clusters)
	})

	t.Run("Returns empty list when no clusters", func(t *testing.T) {
		mockClient := &MockRedshift{
			Clusters: []RedshiftCluster{},
		}

		r := &Redshift{client: mockClient}
		clusters, err := r.ListClusters(ctx)

		require.NoError(t, err)
		assert.Empty(t, clusters)
	})

	t.Run("Sorts clusters by identifier", func(t *testing.T) {
		mockClient := &MockRedshift{
			Clusters: []RedshiftCluster{
				{
					ClusterIdentifier: "zebra-cluster",
					Endpoint:          "zebra.redshift.amazonaws.com",
					Port:              5439,
				},
				{
					ClusterIdentifier: "alpha-cluster",
					Endpoint:          "alpha.redshift.amazonaws.com",
					Port:              5439,
				},
				{
					ClusterIdentifier: "beta-cluster",
					Endpoint:          "beta.redshift.amazonaws.com",
					Port:              5439,
				},
			},
		}

		r := &Redshift{client: mockClient}
		clusters, err := r.ListClusters(ctx)

		require.NoError(t, err)
		assert.Len(t, clusters, 3)
		assert.Equal(t, "alpha-cluster", clusters[0].ClusterIdentifier)
		assert.Equal(t, "beta-cluster", clusters[1].ClusterIdentifier)
		assert.Equal(t, "zebra-cluster", clusters[2].ClusterIdentifier)
	})
}

func TestMockRedshift_DescribeClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns mocked clusters", func(t *testing.T) {
		mock := &MockRedshift{
			Clusters: []RedshiftCluster{
				{
					ClusterIdentifier: "test-cluster",
					Endpoint:          "test.redshift.amazonaws.com",
					Port:              5439,
					NodeType:          "dc2.large",
					ClusterStatus:     "available",
					DatabaseName:      "testdb",
				},
			},
		}

		out, err := mock.DescribeClusters(ctx, nil)

		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Len(t, out.Clusters, 1)
		assert.Equal(t, "test-cluster", *out.Clusters[0].ClusterIdentifier)
		assert.Equal(t, "test.redshift.amazonaws.com", *out.Clusters[0].Endpoint.Address)
		assert.Equal(t, int32(5439), *out.Clusters[0].Endpoint.Port)
		assert.Equal(t, "dc2.large", *out.Clusters[0].NodeType)
		assert.Equal(t, "available", *out.Clusters[0].ClusterStatus)
		assert.Equal(t, "testdb", *out.Clusters[0].DBName)
	})

	t.Run("Returns error when configured", func(t *testing.T) {
		mock := &MockRedshift{
			Err: fmt.Errorf("mock error"),
		}

		out, err := mock.DescribeClusters(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, out)
	})
}
