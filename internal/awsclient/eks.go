package awsclient

import (
	"context"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	eks "github.com/aws/aws-sdk-go-v2/service/eks"
)

// EKSAPI abstracts the subset of the EKS client we use (makes testing easier).
type EKSAPI interface {
	DescribeCluster(ctx context.Context, params *eks.DescribeClusterInput, optFns ...func(*eks.Options)) (*eks.DescribeClusterOutput, error)
}

// EKS wraps the sdk client.
type EKS struct{ client EKSAPI }

func NewEKS(cfg awsv2.Config) *EKS { return &EKS{client: eks.NewFromConfig(cfg)} }

// Cluster models the minimal information we need from an EKS cluster.
type Cluster struct {
	Name     string
	Endpoint string
}

// GetCluster returns the EKS cluster basic info (name, endpoint).
func (e *EKS) GetCluster(ctx context.Context, name string) (Cluster, error) {
	out, err := e.client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &name})
	if err != nil {
		return Cluster{}, err
	}
	if out.Cluster == nil || out.Cluster.Endpoint == nil {
		return Cluster{}, nil
	}
	c := Cluster{Name: name, Endpoint: *out.Cluster.Endpoint}
	return c, nil
}
