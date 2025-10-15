package awsclient

import (
	"context"
	"sort"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	rds "github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// RDSAPI abstraction.
type RDSAPI interface {
	DescribeDBInstances(
		ctx context.Context,
		params *rds.DescribeDBInstancesInput,
		optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error)
}

type RDS struct{ client RDSAPI }

func NewRDS(cfg awsv2.Config) *RDS { return &RDS{client: rds.NewFromConfig(cfg)} }

// Endpoint describes a database endpoint.
type Endpoint struct {
	Address          string
	Port             int
	DBInstanceID     string
	Engine           string
	EngineVersion    string
	DBInstanceStatus string
	DBInstanceClass  string
}

// ListEndpoints lists endpoints of all DB instances.
func (r *RDS) ListEndpoints(ctx context.Context) ([]Endpoint, error) {
	var endpoints []Endpoint
	marker := (*string)(nil)
	for {
		out, err := r.client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, db := range out.DBInstances {
			if db.Endpoint != nil && db.Endpoint.Address != nil && db.Endpoint.Port != nil {
				ep := Endpoint{
					Address: *db.Endpoint.Address,
					Port:    int(*db.Endpoint.Port),
				}
				if db.DBInstanceIdentifier != nil {
					ep.DBInstanceID = *db.DBInstanceIdentifier
				}
				if db.Engine != nil {
					ep.Engine = *db.Engine
				}
				if db.EngineVersion != nil {
					ep.EngineVersion = *db.EngineVersion
				}
				if db.DBInstanceStatus != nil {
					ep.DBInstanceStatus = *db.DBInstanceStatus
				}
				if db.DBInstanceClass != nil {
					ep.DBInstanceClass = *db.DBInstanceClass
				}
				endpoints = append(endpoints, ep)
			}
		}
		if out.Marker == nil {
			break
		}
		marker = out.Marker
	}
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Address == endpoints[j].Address {
			return endpoints[i].Port < endpoints[j].Port
		}
		return endpoints[i].Address < endpoints[j].Address
	})
	return endpoints, nil
}

// Mock for tests.
type MockRDS struct {
	Endpoints []Endpoint
	Err       error
}

func (m *MockRDS) DescribeDBInstances(
	ctx context.Context,
	params *rds.DescribeDBInstancesInput,
	optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	out := &rds.DescribeDBInstancesOutput{}
	for _, ep := range m.Endpoints {
		addr := ep.Address
		port := int32(ep.Port)
		dbid := ep.DBInstanceID
		engine := ep.Engine
		engineVer := ep.EngineVersion
		status := ep.DBInstanceStatus
		class := ep.DBInstanceClass
		out.DBInstances = append(out.DBInstances,
			types.DBInstance{
				Endpoint:             &types.Endpoint{Address: &addr, Port: &port},
				DBInstanceIdentifier: &dbid,
				Engine:               &engine,
				EngineVersion:        &engineVer,
				DBInstanceStatus:     &status,
				DBInstanceClass:      &class,
			})
	}
	return out, nil
}
