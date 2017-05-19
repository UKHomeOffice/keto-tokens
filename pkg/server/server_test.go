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

package server

import (
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	s, err := newFakeServer(newFakeServerConfig())
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

func TestServerStart(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	tk := newFakeTokenProvider()
	c := newFakeProvider(newFakePools())
	cfg := newFakeServerConfig()
	cfg.ReconcileInterval = time.Duration(10) * time.Millisecond

	s, err := New(cfg, c, tk)
	assert.NoError(t, err)
	assert.NotNil(t, s)
	go func() {
		err = s.Start()
		assert.NoError(t, err)
	}()
	<-time.After(time.Duration(100) * time.Millisecond)
	// check nodes are tagged
	pools, _ := c.DescribePools(cfg.Filters)
	for _, p := range pools {
		for _, i := range p.Nodes {
			tag, found, _ := c.GetNodeTag(i, cfg.TagName)
			assert.True(t, found)
			assert.NotEmpty(t, tag)
		}
	}
}

func newFakeServer(cfg Config) (*Server, error) {
	log.SetOutput(ioutil.Discard)
	t := newFakeTokenProvider()
	c := newFakeProvider(newFakePools())

	return New(cfg, c, t)
}

func newFakePools() []cloud.Pool {
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
			Nodes: []cloud.NodeID{"compute00-gp0", "compute01-gp0"},
			Tags: cloud.NodeTags{
				"Role": "compute",
				"Env":  "dev",
			},
		},
		{
			Name:  "compute1",
			Nodes: []cloud.NodeID{"compute00-gp1", "compute01-gp1", "compute02-gp1", "compute02-gp1"},
			Tags: cloud.NodeTags{
				"Role": "compute",
				"Env":  "dev",
			},
		},
	}
}

func newFakeServerConfig() Config {
	return Config{
		MasterAPI: "https://127.0.0.1",
		KubeToken: "token",
		TagName:   "KubeletToken",
		TokenTTL:  time.Duration(10) * time.Minute,
		Filters: cloud.NodeTags{
			"Role": "compute",
			"Env":  "dev",
		},
	}
}

type fakeProvider struct {
	sync.RWMutex
	pools []cloud.Pool
	nodes map[cloud.NodeID]cloud.NodeTags
}

func newFakeProvider(pools []cloud.Pool) cloud.Provider {
	c := &fakeProvider{
		pools: pools,
		nodes: make(map[cloud.NodeID]cloud.NodeTags, 0),
	}
	for _, p := range pools {
		for _, n := range p.Nodes {
			c.nodes[n] = p.Tags.Clone()
		}
	}

	return c
}

func (f *fakeProvider) GetNodeID() (cloud.NodeID, error) {
	return "compute00", nil
}

func (f *fakeProvider) DescribePools(filters cloud.NodeTags) ([]cloud.Pool, error) {
	var list []cloud.Pool
	for _, p := range f.pools {
		found := true
		for k, v := range filters {
			val, exists := p.Tags[k]
			if !exists || val != v {
				found = false
				break
			}
		}
		if found {
			list = append(list, p)
		}
	}

	return list, nil
}

// GetNodeTags retrieves a list of node tags
func (f *fakeProvider) GetNodeTags(id cloud.NodeID) (cloud.NodeTags, error) {
	f.RLock()
	defer f.RUnlock()
	if n, found := f.nodes[id]; found {
		return n, nil
	}

	return cloud.NodeTags{}, cloud.ErrInstanceNotFound
}

// GetNodeTag retrieves a specific node tag
func (f *fakeProvider) GetNodeTag(id cloud.NodeID, tag string) (string, bool, error) {
	f.RLock()
	defer f.RUnlock()

	n, err := f.GetNodeTags(id)
	if err != nil {
		return "", false, err
	}
	v, found := n[tag]

	return v, found, nil
}

// SetNodeTags is used to set a series of tags on a node
func (f *fakeProvider) SetNodeTags(id cloud.NodeID, tags cloud.NodeTags) error {
	f.Lock()
	defer f.Unlock()

	t := f.nodes[id]
	for k, v := range tags {
		t[k] = v
	}
	f.nodes[id] = t

	return nil
}
