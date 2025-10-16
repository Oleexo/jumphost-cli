package connect

import (
	"context"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
	"github.com/Oleexo/jumphost-cli/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// Function variables for testing - these can be overridden in tests

var LoadConfigFunc = awsclient.LoadDefaultConfig
var ValidateCredentialsFunc = awsclient.ValidateCredentials
var GetIdentityFunc = awsclient.GetIdentity
var NewEC2Func = func(cfg aws.Config) awsclient.EC2 { return awsclient.NewEC2(cfg) }
var NewRDSFunc = func(cfg aws.Config) *awsclient.RDS { return awsclient.NewRDS(cfg) }
var NewEKSFunc = func(cfg aws.Config) *awsclient.EKS { return awsclient.NewEKS(cfg) }

// ListJumpHostInstancesFn allows mocking EC2 list operations
var ListJumpHostInstancesFn = func(e awsclient.EC2, ctx context.Context, tag string) (
	[]models.JumphostInstance,
	error) {
	return e.ListJumpHostInstances(ctx, tag)
}

// GetEKSClusterFn allows mocking EKS operations
var GetEKSClusterFn = func(e *awsclient.EKS, ctx context.Context, name string) (awsclient.Cluster, error) {
	return e.GetCluster(ctx, name)
}

// StartPortForwarding is a placeholder for port forwarding operations
var StartPortForwarding = func(
	ctx context.Context,
	instanceID, host string,
	remotePort, localPort int,
	region string) error {
	return nil
}
