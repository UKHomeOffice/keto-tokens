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
	"testing"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	awsp "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
)

func TestDescribePools(t *testing.T) {
	cs := []struct {
		Tags cloud.NodeTags
		Size int
	}{
		{Tags: cloud.NodeTags{"Env": "dev"}, Size: 3},
		{Tags: cloud.NodeTags{"Role": "compute"}, Size: 3},
		{Tags: cloud.NodeTags{"Role": "compute", "Env": "dev"}, Size: 2},
		{Tags: cloud.NodeTags{"Role": "master"}, Size: 1},
	}
	p := newFakeAWS(newFakeSetup())
	for i, c := range cs {
		g, err := p.DescribePools(c.Tags)
		assert.NoError(t, err, "case %d should not have thrown error", i)
		if !assert.NotNil(t, g, "case %d should not be nil", i) {
			continue
		}
		assert.Equal(t, c.Size, len(g), "case %d, expected: %d, got: %d", i, c.Size, len(g))
	}
}

func TestDescribePoolsEmpty(t *testing.T) {
	p := newFakeAWS(newFakeSetup())
	groups, err := p.DescribePools(nil)
	assert.NoError(t, err)
	assert.NotNil(t, groups)
	assert.NotEmpty(t, groups)
	assert.Equal(t, len(newFakeSetup()), len(groups))
}

func TestDescribePoolsByFilter(t *testing.T) {
	p := newFakeAWS(newFakeSetup())
	groups, err := p.DescribePools(cloud.NodeTags{
		"Role": "master",
	})
	assert.NoError(t, err)
	assert.NotNil(t, groups)
	assert.NotEmpty(t, groups)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, "masters", groups[0].Name)
}

func TestGetNodeTags(t *testing.T) {
	p := newFakeAWS(newFakeSetup())
	tags, err := p.GetNodeTags("compute00")
	assert.NoError(t, err)
	assert.NotEmpty(t, tags)
}

func TestGetNodeTagsNotFound(t *testing.T) {
	p := newFakeAWS(newFakeSetup())
	tags, err := p.GetNodeTags("not_there")
	assert.Error(t, err)
	assert.Empty(t, tags)
	assert.Equal(t, err.Error(), cloud.ErrInstanceNotFound.Error())
}

func TestGetNodeTag(t *testing.T) {
	cs := []struct {
		ID       cloud.NodeID
		Tag      string
		Expected string
		NoError  bool
	}{
		{},
		{
			ID: "not_there",
		},
		{
			ID:      "compute00",
			Tag:     "not_there",
			NoError: true,
		},
		{
			ID:       "compute00",
			Tag:      "Role",
			Expected: "compute",
			NoError:  true,
		},
	}
	p := newFakeAWS(newFakeSetup())
	for i, c := range cs {
		v, found, err := p.GetNodeTag(c.ID, c.Tag)
		if !c.NoError && err == nil {
			t.Errorf("case %d should not have thrown error: %s", i, err)
			continue
		}
		if c.Expected != "" {
			assert.True(t, found, "case %d should be true", i)
			assert.Equal(t, c.Expected, v, "case %d, expected: %s, got: %s", i, c.Expected, v)
		} else {
			assert.False(t, found, "case %d should be false", i)
		}
	}
}

func TestSetNodeTag(t *testing.T) {
	cs := []struct {
		ID   cloud.NodeID
		Tags cloud.NodeTags
		Ok   bool
	}{
		{},
		{ID: "not_there"},
		{ID: "compute00", Ok: true},
		{ID: "compute00", Tags: cloud.NodeTags{"Test": "Tag"}},
	}
	p := newFakeAWS(newFakeSetup())
	for i, c := range cs {
		err := p.SetNodeTags(c.ID, c.Tags)
		if !c.Ok && err != nil {
			t.Errorf("case %d should have thrown error", i)
			continue
		}
		assert.NoError(t, err)
	}
}

func newFakeSetup() []cloud.Pool {
	return []cloud.Pool{
		{
			Name:  "masters",
			Nodes: []cloud.NodeID{"master0", "master1"},
			Tags: cloud.NodeTags{
				"Role": "master",
				"Env":  "dev",
			},
		},
		{
			Name:  "compute0",
			Nodes: []cloud.NodeID{"compute00", "compute11"},
			Tags: cloud.NodeTags{
				"Role": "compute",
				"Env":  "dev",
			},
		},
		{
			Name:  "compute1",
			Nodes: []cloud.NodeID{"compute10", "compute11", "compute12", "compute13"},
			Tags: cloud.NodeTags{
				"Role": "compute",
				"Env":  "dev",
			},
		},
		{
			Name:  "other_compute",
			Nodes: []cloud.NodeID{"compute10", "compute11", "compute12", "compute13"},
			Tags: cloud.NodeTags{
				"Role": "compute",
				"Env":  "other_env",
			},
		},
	}
}

func newFakeAWS(pools []cloud.Pool) *awsProvider {
	scale := &fakeAutoscalingProvider{
		pools: pools,
	}
	compute := &fakeComputeProvider{
		nodes: make(map[cloud.NodeID]cloud.NodeTags),
	}
	c := &awsProvider{client: scale, compute: compute}
	// let populate the nodes
	for _, p := range pools {
		for _, x := range p.Nodes {
			compute.nodes[x] = p.Tags.Clone()
		}
	}

	return c
}

type fakeAutoscalingProvider struct {
	autoscalingiface.AutoScalingAPI
	pools []cloud.Pool
}

func (f *fakeAutoscalingProvider) DescribeAutoScalingGroups(input *autoscaling.DescribeAutoScalingGroupsInput) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	resp := &autoscaling.DescribeAutoScalingGroupsOutput{
		AutoScalingGroups: make([]*autoscaling.Group, 0),
	}
	for _, x := range f.pools {
		group := &autoscaling.Group{
			AutoScalingGroupName: awsp.String(x.Name),
			Tags:                 make([]*autoscaling.TagDescription, 0),
			Instances:            make([]*autoscaling.Instance, 0),
		}
		for k, v := range x.Tags {
			group.Tags = append(group.Tags, &autoscaling.TagDescription{
				Key:   awsp.String(k),
				Value: awsp.String(v),
			})
		}
		for _, i := range x.Nodes {
			group.Instances = append(group.Instances, &autoscaling.Instance{
				InstanceId: awsp.String(string(i)),
			})
		}

		resp.AutoScalingGroups = append(resp.AutoScalingGroups, group)
	}

	return resp, nil
}

type fakeComputeProvider struct {
	ec2iface.EC2API
	nodes map[cloud.NodeID]cloud.NodeTags
}

func (f *fakeComputeProvider) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	instances := make([]*ec2.Instance, 0)
	if input.InstanceIds != nil && len(input.InstanceIds) > 0 {
		for _, id := range input.InstanceIds {
			nodeID := cloud.NodeID(*id)
			if n, found := f.nodes[nodeID]; found {
				instances = append(instances, f.convertToInstance(nodeID, n))
			}
		}
	} else {
		for id, n := range f.nodes {
			instances = append(instances, f.convertToInstance(id, n))
		}
	}

	return &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			{Instances: instances},
		},
	}, nil
}

func (f *fakeComputeProvider) CreateTags(input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	for _, r := range input.Resources {
		nodeID := cloud.NodeID(*r)
		if node, found := f.nodes[nodeID]; found {
			for _, t := range input.Tags {
				node[*t.Key] = *t.Value
			}
		}
	}

	return nil, nil
}

func (f *fakeComputeProvider) convertToInstance(id cloud.NodeID, tags cloud.NodeTags) *ec2.Instance {
	in := &ec2.Instance{
		InstanceId: awsp.String(string(id)),
		Tags:       make([]*ec2.Tag, 0),
	}
	for k, v := range tags {
		in.Tags = append(in.Tags, &ec2.Tag{
			Key:   awsp.String(k),
			Value: awsp.String(v),
		})
	}

	return in
}
