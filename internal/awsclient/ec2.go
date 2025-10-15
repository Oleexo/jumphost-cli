package awsclient

import (
	"context"
	"errors"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// EC2API abstracts the subset of the EC2 client we use (makes testing easier).
type EC2API interface {
	DescribeInstances(
		ctx context.Context,
		params *awsec2.DescribeInstancesInput,
		optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error)
}

// EC2 wraps the sdk client.
type EC2 struct{ client EC2API }

func NewEC2(cfg aws.Config) *EC2 { return &EC2{client: awsec2.NewFromConfig(cfg)} }

// ListJumpHostInstances returns all running instance IDs with tag Usage=tagValue (sorted).
func (e *EC2) ListJumpHostInstances(ctx context.Context, tagValue string) ([]string, error) {
	out, err := e.client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: aws.String("tag:Usage"), Values: []string{tagValue}},
			{Name: aws.String("instance-state-name"), Values: []string{"running"}},
		},
	})
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if inst.InstanceId != nil {
				ids = append(ids, *inst.InstanceId)
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// GetFirstJumpHostInstance returns first instance id tagged Usage=tagValue in running state (legacy helper).
func (e *EC2) GetFirstJumpHostInstance(ctx context.Context, tagValue string) (string, error) {
	ids, err := e.ListJumpHostInstances(ctx, tagValue)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	return ids[0], nil
}

// Mock for tests.
type MockEC2 struct {
	IDs []string
	Err error
}

func (m *MockEC2) DescribeInstances(
	ctx context.Context,
	params *awsec2.DescribeInstancesInput,
	optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if len(m.IDs) == 0 {
		return &awsec2.DescribeInstancesOutput{}, nil
	}
	out := &awsec2.DescribeInstancesOutput{Reservations: []types.Reservation{{Instances: []types.Instance{}}}}
	for _, id := range m.IDs {
		out.Reservations[0].Instances = append(out.Reservations[0].Instances,
			types.Instance{InstanceId: aws.String(id)})
	}
	return out, nil
}

// Helper to simulate an error.
var ErrSimulated = errors.New("simulated error")
