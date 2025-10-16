package awsclient

import (
	"context"
	"sort"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
)

// ElastiCacheAPI abstraction.
type ElastiCacheAPI interface {
	DescribeCacheClusters(
		ctx context.Context,
		params *elasticache.DescribeCacheClustersInput,
		optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
	DescribeReplicationGroups(
		ctx context.Context,
		params *elasticache.DescribeReplicationGroupsInput,
		optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error)
}

type Redis struct{ client ElastiCacheAPI }

func NewRedis(cfg awsv2.Config) *Redis {
	return &Redis{client: elasticache.NewFromConfig(cfg)}
}

// RedisEndpoint describes a Redis/ElastiCache endpoint.
type RedisEndpoint struct {
	Name           string
	Endpoint       string
	Port           int
	Engine         string
	EngineVersion  string
	Status         string
	NodeType       string
	IsCluster      bool
	ReaderEndpoint string
}

// ListEndpoints lists endpoints of all Redis/ElastiCache clusters and replication groups.
func (r *Redis) ListEndpoints(ctx context.Context) ([]RedisEndpoint, error) {
	var endpoints []RedisEndpoint

	// Get standalone cache clusters
	clusterMarker := (*string)(nil)
	for {
		out, err := r.client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
			Marker:                                  clusterMarker,
			ShowCacheNodeInfo:                       awsv2.Bool(true),
			ShowCacheClustersNotInReplicationGroups: awsv2.Bool(true),
		})
		if err != nil {
			return nil, err
		}

		for _, cluster := range out.CacheClusters {
			// Skip clusters that are part of replication groups (we'll get them separately)
			if cluster.ReplicationGroupId != nil && *cluster.ReplicationGroupId != "" {
				continue
			}

			if cluster.CacheNodes != nil && len(cluster.CacheNodes) > 0 {
				node := cluster.CacheNodes[0]
				if node.Endpoint != nil && node.Endpoint.Address != nil && node.Endpoint.Port != nil {
					ep := RedisEndpoint{
						Endpoint: *node.Endpoint.Address,
						Port:     int(*node.Endpoint.Port),
					}
					if cluster.CacheClusterId != nil {
						ep.Name = *cluster.CacheClusterId
					}
					if cluster.Engine != nil {
						ep.Engine = *cluster.Engine
					}
					if cluster.EngineVersion != nil {
						ep.EngineVersion = *cluster.EngineVersion
					}
					if cluster.CacheClusterStatus != nil {
						ep.Status = *cluster.CacheClusterStatus
					}
					if cluster.CacheNodeType != nil {
						ep.NodeType = *cluster.CacheNodeType
					}
					endpoints = append(endpoints, ep)
				}
			}
		}

		if out.Marker == nil {
			break
		}
		clusterMarker = out.Marker
	}

	// Get replication groups (for Redis clusters)
	rgMarker := (*string)(nil)
	for {
		out, err := r.client.DescribeReplicationGroups(ctx, &elasticache.DescribeReplicationGroupsInput{
			Marker: rgMarker,
		})
		if err != nil {
			return nil, err
		}

		for _, rg := range out.ReplicationGroups {
			var ep RedisEndpoint
			if rg.ReplicationGroupId != nil {
				ep.Name = *rg.ReplicationGroupId
			}
			if rg.Status != nil {
				ep.Status = *rg.Status
			}
			ep.IsCluster = true

			// Primary endpoint
			if rg.NodeGroups != nil && len(rg.NodeGroups) > 0 {
				ng := rg.NodeGroups[0]
				if ng.PrimaryEndpoint != nil && ng.PrimaryEndpoint.Address != nil && ng.PrimaryEndpoint.Port != nil {
					ep.Endpoint = *ng.PrimaryEndpoint.Address
					ep.Port = int(*ng.PrimaryEndpoint.Port)

					// Reader endpoint
					if ng.ReaderEndpoint != nil && ng.ReaderEndpoint.Address != nil {
						ep.ReaderEndpoint = *ng.ReaderEndpoint.Address
					}
				}
			}

			// Get engine info from member clusters
			if len(rg.MemberClusters) > 0 {
				clusterID := rg.MemberClusters[0]
				clusterOut, err := r.client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
					CacheClusterId: &clusterID,
				})
				if err == nil && len(clusterOut.CacheClusters) > 0 {
					cluster := clusterOut.CacheClusters[0]
					if cluster.Engine != nil {
						ep.Engine = *cluster.Engine
					}
					if cluster.EngineVersion != nil {
						ep.EngineVersion = *cluster.EngineVersion
					}
					if cluster.CacheNodeType != nil {
						ep.NodeType = *cluster.CacheNodeType
					}
				}
			}

			if ep.Endpoint != "" {
				endpoints = append(endpoints, ep)
			}
		}

		if out.Marker == nil {
			break
		}
		rgMarker = out.Marker
	}

	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].Name < endpoints[j].Name
	})

	return endpoints, nil
}

// Mock for tests.
type MockRedis struct {
	Endpoints []RedisEndpoint
	Err       error
}

func (m *MockRedis) DescribeCacheClusters(
	ctx context.Context,
	params *elasticache.DescribeCacheClustersInput,
	optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	out := &elasticache.DescribeCacheClustersOutput{}
	for _, ep := range m.Endpoints {
		if ep.IsCluster {
			continue // Skip replication groups
		}
		addr := ep.Endpoint
		port := int32(ep.Port)
		id := ep.Name
		engine := ep.Engine
		engineVer := ep.EngineVersion
		status := ep.Status
		nodeType := ep.NodeType
		out.CacheClusters = append(out.CacheClusters, types.CacheCluster{
			CacheClusterId:     &id,
			CacheClusterStatus: &status,
			Engine:             &engine,
			EngineVersion:      &engineVer,
			CacheNodeType:      &nodeType,
			CacheNodes: []types.CacheNode{
				{
					Endpoint: &types.Endpoint{
						Address: &addr,
						Port:    &port,
					},
				},
			},
		})
	}
	return out, nil
}

func (m *MockRedis) DescribeReplicationGroups(
	ctx context.Context,
	params *elasticache.DescribeReplicationGroupsInput,
	optFns ...func(*elasticache.Options)) (*elasticache.DescribeReplicationGroupsOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	out := &elasticache.DescribeReplicationGroupsOutput{}
	for _, ep := range m.Endpoints {
		if !ep.IsCluster {
			continue // Skip standalone clusters
		}
		addr := ep.Endpoint
		port := int32(ep.Port)
		id := ep.Name
		status := ep.Status
		out.ReplicationGroups = append(out.ReplicationGroups, types.ReplicationGroup{
			ReplicationGroupId: &id,
			Status:             &status,
			NodeGroups: []types.NodeGroup{
				{
					PrimaryEndpoint: &types.Endpoint{
						Address: &addr,
						Port:    &port,
					},
				},
			},
		})
	}
	return out, nil
}
