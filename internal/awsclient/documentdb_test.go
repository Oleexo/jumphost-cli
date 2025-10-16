package awsclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocumentDB(t *testing.T) {
	// Just verify construction doesn't panic
	// Can't fully test without valid AWS config
	// db := NewDocumentDB(aws.Config{})
	// assert.NotNil(t, db)
}

func TestDocumentDB_ListClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns clusters successfully", func(t *testing.T) {
		mockClient := &MockDocumentDB{
			Clusters: []DBCluster{
				{
					ClusterIdentifier: "cluster-1",
					Endpoint:          "cluster-1.docdb.amazonaws.com",
					Port:              27017,
					Engine:            "docdb",
					EngineVersion:     "4.0.0",
					Status:            "available",
					ReaderEndpoint:    "cluster-1-ro.docdb.amazonaws.com",
				},
				{
					ClusterIdentifier: "cluster-2",
					Endpoint:          "cluster-2.docdb.amazonaws.com",
					Port:              27017,
					Engine:            "docdb",
					EngineVersion:     "5.0.0",
					Status:            "available",
					ReaderEndpoint:    "cluster-2-ro.docdb.amazonaws.com",
				},
			},
		}

		db := &DocumentDB{client: mockClient}
		clusters, err := db.ListClusters(ctx)

		require.NoError(t, err)
		assert.Len(t, clusters, 2)
		assert.Equal(t, "cluster-1", clusters[0].ClusterIdentifier)
		assert.Equal(t, "cluster-1.docdb.amazonaws.com", clusters[0].Endpoint)
		assert.Equal(t, 27017, clusters[0].Port)
		assert.Equal(t, "docdb", clusters[0].Engine)
		assert.Equal(t, "4.0.0", clusters[0].EngineVersion)
		assert.Equal(t, "available", clusters[0].Status)
		assert.Equal(t, "cluster-1-ro.docdb.amazonaws.com", clusters[0].ReaderEndpoint)
	})

	t.Run("Returns error when API fails", func(t *testing.T) {
		mockClient := &MockDocumentDB{
			Err: fmt.Errorf("API error"),
		}

		db := &DocumentDB{client: mockClient}
		clusters, err := db.ListClusters(ctx)

		assert.Error(t, err)
		assert.Nil(t, clusters)
	})

	t.Run("Returns empty list when no clusters", func(t *testing.T) {
		mockClient := &MockDocumentDB{
			Clusters: []DBCluster{},
		}

		db := &DocumentDB{client: mockClient}
		clusters, err := db.ListClusters(ctx)

		require.NoError(t, err)
		assert.Empty(t, clusters)
	})

	t.Run("Sorts clusters by identifier", func(t *testing.T) {
		mockClient := &MockDocumentDB{
			Clusters: []DBCluster{
				{
					ClusterIdentifier: "zebra-cluster",
					Endpoint:          "zebra.docdb.amazonaws.com",
					Port:              27017,
				},
				{
					ClusterIdentifier: "alpha-cluster",
					Endpoint:          "alpha.docdb.amazonaws.com",
					Port:              27017,
				},
				{
					ClusterIdentifier: "beta-cluster",
					Endpoint:          "beta.docdb.amazonaws.com",
					Port:              27017,
				},
			},
		}

		db := &DocumentDB{client: mockClient}
		clusters, err := db.ListClusters(ctx)

		require.NoError(t, err)
		assert.Len(t, clusters, 3)
		assert.Equal(t, "alpha-cluster", clusters[0].ClusterIdentifier)
		assert.Equal(t, "beta-cluster", clusters[1].ClusterIdentifier)
		assert.Equal(t, "zebra-cluster", clusters[2].ClusterIdentifier)
	})
}

func TestMockDocumentDB_DescribeDBClusters(t *testing.T) {
	ctx := context.Background()

	t.Run("Returns mocked clusters", func(t *testing.T) {
		mock := &MockDocumentDB{
			Clusters: []DBCluster{
				{
					ClusterIdentifier: "test-cluster",
					Endpoint:          "test.docdb.amazonaws.com",
					Port:              27017,
					Engine:            "docdb",
					EngineVersion:     "4.0.0",
					Status:            "available",
					ReaderEndpoint:    "test-ro.docdb.amazonaws.com",
				},
			},
		}

		out, err := mock.DescribeDBClusters(ctx, nil)

		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Len(t, out.DBClusters, 1)
		assert.Equal(t, "test-cluster", *out.DBClusters[0].DBClusterIdentifier)
		assert.Equal(t, "test.docdb.amazonaws.com", *out.DBClusters[0].Endpoint)
		assert.Equal(t, int32(27017), *out.DBClusters[0].Port)
	})

	t.Run("Returns error when configured", func(t *testing.T) {
		mock := &MockDocumentDB{
			Err: fmt.Errorf("mock error"),
		}

		out, err := mock.DescribeDBClusters(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, out)
	})
}
