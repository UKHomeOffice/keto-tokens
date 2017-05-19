/*
Copyright 2017 The Keto Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"os"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	awsp "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type awsProvider struct {
	client   autoscalingiface.AutoScalingAPI
	compute  ec2iface.EC2API
	metadata *ec2metadata.EC2InstanceIdentityDocument
}

type awsPlugin struct{}

func init() {
	cloud.Register("aws", &awsPlugin{})
}

// New creates a new aws cloud provider
func (r awsPlugin) New() (cloud.Provider, error) {
	// step: attempt to get the region
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		m, err := getInstanceMetadata()
		if err != nil {
			return nil, err
		}
		region = m.Region
	}
	cfg := &awsp.Config{Region: awsp.String(region)}
	compute := ec2.New(session.New(), cfg)
	asg := autoscaling.New(session.New(), cfg)

	return &awsProvider{
		client:  asg,
		compute: compute,
	}, nil
}

func getInstanceMetadata() (ec2metadata.EC2InstanceIdentityDocument, error) {
	session := session.New()
	client := ec2metadata.New(session)
	data, err := client.GetInstanceIdentityDocument()
	if err != nil {
		return ec2metadata.EC2InstanceIdentityDocument{}, err
	}

	return data, nil
}

// GetNodeID returns our node id
func (a *awsProvider) GetNodeID() (cloud.NodeID, error) {
	if a.metadata == nil {
		doc, err := getInstanceMetadata()
		if err != nil {
			return "", err
		}
		a.metadata = &doc
	}

	return cloud.NodeID(a.metadata.InstanceID), nil
}

// DescribePools is used to retrieve a list of node pools, filters if required by tags
func (a *awsProvider) DescribePools(filter cloud.NodeTags) ([]cloud.Pool, error) {
	groups, err := a.getFilterGroups(filter)
	if err != nil {
		return []cloud.Pool{}, err
	}

	var pools []cloud.Pool
	for _, x := range groups {
		pool := cloud.Pool{
			Name: *x.AutoScalingGroupName,
			Tags: make(cloud.NodeTags, 0),
		}
		for _, i := range x.Instances {
			pool.Nodes = append(pool.Nodes, cloud.NodeID(*i.InstanceId))
		}
		for _, t := range x.Tags {
			if t.Key != nil && t.Value != nil {
				pool.Tags[*t.Key] = *t.Value
			}
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// GetNodeTags retrieves a list of tags for a specific node
func (a *awsProvider) GetNodeTags(id cloud.NodeID) (cloud.NodeTags, error) {
	resp, err := a.compute.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{awsp.String(string(id))},
	})
	if err != nil {
		return cloud.NodeTags{}, err
	}

	tags := make(cloud.NodeTags, 0)
	if resp.Reservations == nil {
		return tags, cloud.ErrInstanceNotFound
	}
	if resp.Reservations[0].Instances == nil || len(resp.Reservations[0].Instances) <= 0 {
		return tags, cloud.ErrInstanceNotFound
	}

	i := resp.Reservations[0].Instances[0]
	for _, t := range i.Tags {
		if t.Key == nil || t.Value == nil {
			continue
		}
		tags[*t.Key] = *t.Value
	}

	return tags, nil
}

// GetNodeTag retrieves a specific instance tag
func (a *awsProvider) GetNodeTag(id cloud.NodeID, tag string) (string, bool, error) {
	tags, err := a.GetNodeTags(id)
	if err != nil {
		return "", false, err
	}

	for k, v := range tags {
		if k == tag {
			return v, true, nil
		}
	}

	return "", false, nil
}

// SetNodeTags updates the tags of a instance
func (a *awsProvider) SetNodeTags(nodeID cloud.NodeID, tags cloud.NodeTags) error {
	if len(tags) < 0 {
		return nil
	}

	var newTags []*ec2.Tag
	for k, v := range tags {
		newTags = append(newTags, &ec2.Tag{
			Key:   awsp.String(k),
			Value: awsp.String(v),
		})
	}

	// step: attempting to update the tags
	_, err := a.compute.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{awsp.String(string(nodeID))},
		Tags:      newTags,
	})

	return err
}

// getFiltersGroups retrieves a list of auto-scaling groups and applies the filter. For some
// god-forsaken reason you cannot search by tags
func (a *awsProvider) getFilterGroups(filter cloud.NodeTags) ([]*autoscaling.Group, error) {
	resp, err := a.client.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return nil, err
	}

	// if we are not filtering
	if len(filter) <= 0 {
		return resp.AutoScalingGroups, nil
	}

	// step: filter out the groups
	var list []*autoscaling.Group
	for _, x := range resp.AutoScalingGroups {
		if filterGroupByTags(filter, x.Tags) {
			list = append(list, x)
		}
	}

	return list, nil
}

// filterGroupByTags checks the group has all the required tags
func filterGroupByTags(filter cloud.NodeTags, tags []*autoscaling.TagDescription) bool {
	count := 0
	for k, v := range filter {
		for _, tag := range tags {
			if tag == nil || tag.Value == nil {
				continue
			}
			if *tag.Key == k && *tag.Value == v {
				count++
			}
		}
	}

	return count == len(filter)
}
