package awsclient

import (
	"context"
	"sort"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	"github.com/aws/aws-sdk-go-v2/service/docdb/types"
)

// DocumentDBAPI abstraction.
type DocumentDBAPI interface {
	DescribeDBClusters(
		ctx context.Context,
		params *docdb.DescribeDBClustersInput,
		optFns ...func(*docdb.Options)) (*docdb.DescribeDBClustersOutput, error)
}

type DocumentDB struct{ client DocumentDBAPI }

func NewDocumentDB(cfg awsv2.Config) *DocumentDB {
	return &DocumentDB{client: docdb.NewFromConfig(cfg)}
}

// DBCluster describes a DocumentDB cluster endpoint.
type DBCluster struct {
	ClusterIdentifier string
	Endpoint          string
	Port              int
	Engine            string
	EngineVersion     string
	Status            string
	ReaderEndpoint    string
}

// ListClusters lists endpoints of all DocumentDB clusters.
func (d *DocumentDB) ListClusters(ctx context.Context) ([]DBCluster, error) {
	var clusters []DBCluster
	marker := (*string)(nil)

	for {
		out, err := d.client.DescribeDBClusters(ctx, &docdb.DescribeDBClustersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		for _, cluster := range out.DBClusters {
			if cluster.Endpoint != nil && cluster.Port != nil {
				c := DBCluster{
					Endpoint: *cluster.Endpoint,
					Port:     int(*cluster.Port),
				}
				if cluster.DBClusterIdentifier != nil {
					c.ClusterIdentifier = *cluster.DBClusterIdentifier
				}
				if cluster.Engine != nil {
					c.Engine = *cluster.Engine
				}
				if cluster.EngineVersion != nil {
					c.EngineVersion = *cluster.EngineVersion
				}
				if cluster.Status != nil {
					c.Status = *cluster.Status
				}
				if cluster.ReaderEndpoint != nil {
					c.ReaderEndpoint = *cluster.ReaderEndpoint
				}
				clusters = append(clusters, c)
			}
		}

		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].ClusterIdentifier < clusters[j].ClusterIdentifier
	})

	return clusters, nil
}

// Mock for tests.
type MockDocumentDB struct {
	Clusters []DBCluster
	Err      error
}

func (m *MockDocumentDB) DescribeDBClusters(
	ctx context.Context,
	params *docdb.DescribeDBClustersInput,
	optFns ...func(*docdb.Options)) (*docdb.DescribeDBClustersOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	out := &docdb.DescribeDBClustersOutput{}
	for _, c := range m.Clusters {
		endpoint := c.Endpoint
		port := int32(c.Port)
		id := c.ClusterIdentifier
		engine := c.Engine
		engineVer := c.EngineVersion
		status := c.Status
		readerEndpoint := c.ReaderEndpoint
		out.DBClusters = append(out.DBClusters, types.DBCluster{
			DBClusterIdentifier: &id,
			Endpoint:            &endpoint,
			Port:                &port,
			Engine:              &engine,
			EngineVersion:       &engineVer,
			Status:              &status,
			ReaderEndpoint:      &readerEndpoint,
		})
	}
	return out, nil
}
