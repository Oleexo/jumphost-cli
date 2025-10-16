package awsclient

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsec2 "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/Oleexo/jumphost-cli/internal/models"
)

// EC2API abstracts the subset of the EC2 client we use (makes testing easier).
type EC2API interface {
	DescribeInstances(
		ctx context.Context,
		params *awsec2.DescribeInstancesInput,
		optFns ...func(*awsec2.Options)) (*awsec2.DescribeInstancesOutput, error)
}

type EC2 interface {
	ListJumpHostInstances(ctx context.Context, tagValue string) ([]models.JumphostInstance, error)
	GetJumpHostInstance(ctx context.Context, instanceID string) (models.JumphostInstance, error)
}
type ec2Impl struct {
	client EC2API
}

func NewEC2(cfg aws.Config) EC2 {
	return &ec2Impl{
		client: awsec2.NewFromConfig(cfg),
	}
}

// ListJumpHostInstances returns all running instance IDs with tag Usage=tagValue (sorted).
func (e *ec2Impl) ListJumpHostInstances(ctx context.Context, tagValue string) ([]models.JumphostInstance, error) {
	out, err := e.client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{Name: aws.String("tag:Usage"), Values: []string{tagValue}},
			{Name: aws.String("instance-state-name"), Values: []string{"running"}},
		},
	})
	if err != nil {
		return nil, err
	}
	var instances []models.JumphostInstance
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if inst.InstanceId == nil {
				continue
			}

			// Find the Name tag
			name := ""
			for _, tag := range inst.Tags {
				if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
					name = *tag.Value
					break
				}
			}

			privateIP := ""
			if inst.PrivateIpAddress != nil {
				privateIP = *inst.PrivateIpAddress
			}

			state := ""
			if inst.State != nil && inst.State.Name != "" {
				state = string(inst.State.Name)
			}

			instances = append(instances, models.JumphostInstance{
				InstanceID:       *inst.InstanceId,
				Name:             name,
				PrivateIPAddress: privateIP,
				State:            state,
			})
		}
	}

	// Sort by Name, then by InstanceID
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].Name != instances[j].Name {
			return instances[i].Name < instances[j].Name
		}
		return instances[i].InstanceID < instances[j].InstanceID
	})

	return instances, nil
}

func (e *ec2Impl) GetJumpHostInstance(ctx context.Context, instanceID string) (models.JumphostInstance, error) {
	out, err := e.client.DescribeInstances(ctx, &awsec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return models.JumphostInstance{}, err
	}

	// Find the instance in the response
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if inst.InstanceId == nil || *inst.InstanceId != instanceID {
				continue
			}

			// Find the Name tag
			name := ""
			for _, tag := range inst.Tags {
				if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
					name = *tag.Value
					break
				}
			}

			privateIP := ""
			if inst.PrivateIpAddress != nil {
				privateIP = *inst.PrivateIpAddress
			}

			state := ""
			if inst.State != nil && inst.State.Name != "" {
				state = string(inst.State.Name)
			}

			return models.JumphostInstance{
				InstanceID:       *inst.InstanceId,
				Name:             name,
				PrivateIPAddress: privateIP,
				State:            state,
			}, nil
		}
	}

	return models.JumphostInstance{}, err
}
