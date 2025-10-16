package awsclient

import (
	"context"
	"sort"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
)

// RedshiftAPI abstraction.
type RedshiftAPI interface {
	DescribeClusters(
		ctx context.Context,
		params *redshift.DescribeClustersInput,
		optFns ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error)
}

type Redshift struct{ client RedshiftAPI }

func NewRedshift(cfg awsv2.Config) *Redshift {
	return &Redshift{client: redshift.NewFromConfig(cfg)}
}

// Cluster describes a Redshift cluster endpoint.
type RedshiftCluster struct {
	ClusterIdentifier string
	Endpoint          string
	Port              int
	NodeType          string
	ClusterStatus     string
	DatabaseName      string
}

// ListClusters lists endpoints of all Redshift clusters.
func (r *Redshift) ListClusters(ctx context.Context) ([]RedshiftCluster, error) {
	var clusters []RedshiftCluster
	marker := (*string)(nil)

	for {
		out, err := r.client.DescribeClusters(ctx, &redshift.DescribeClustersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		for _, cluster := range out.Clusters {
			if cluster.Endpoint != nil && cluster.Endpoint.Address != nil && cluster.Endpoint.Port != nil {
				c := RedshiftCluster{
					Endpoint: *cluster.Endpoint.Address,
					Port:     int(*cluster.Endpoint.Port),
				}
				if cluster.ClusterIdentifier != nil {
					c.ClusterIdentifier = *cluster.ClusterIdentifier
				}
				if cluster.NodeType != nil {
					c.NodeType = *cluster.NodeType
				}
				if cluster.ClusterStatus != nil {
					c.ClusterStatus = *cluster.ClusterStatus
				}
				if cluster.DBName != nil {
					c.DatabaseName = *cluster.DBName
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
type MockRedshift struct {
	Clusters []RedshiftCluster
	Err      error
}

func (m *MockRedshift) DescribeClusters(
	ctx context.Context,
	params *redshift.DescribeClustersInput,
	optFns ...func(*redshift.Options)) (*redshift.DescribeClustersOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	out := &redshift.DescribeClustersOutput{}
	for _, c := range m.Clusters {
		addr := c.Endpoint
		port := int32(c.Port)
		id := c.ClusterIdentifier
		nodeType := c.NodeType
		status := c.ClusterStatus
		dbName := c.DatabaseName
		out.Clusters = append(out.Clusters, types.Cluster{
			Endpoint: &types.Endpoint{
				Address: &addr,
				Port:    &port,
			},
			ClusterIdentifier: &id,
			NodeType:          &nodeType,
			ClusterStatus:     &status,
			DBName:            &dbName,
		})
	}
	return out, nil
}
