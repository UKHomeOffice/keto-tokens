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

package cloud

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrInstanceNotFound was not found
	ErrInstanceNotFound = errors.New("no instances found")
)

// Pool is a collection of compute nodes
type Pool struct {
	// Name is the name of the node pool
	Name string
	// Nodes if a collection of node ids
	Nodes []NodeID
	// Tags is a collection of tags on the node pool
	Tags NodeTags
}

// NodeID is a light-weight wrapper to a node
type NodeID string

// NodeTags is a collection of node tags
type NodeTags map[string]string

func (n *NodeTags) String() string {
	var keys, list []string
	for k := range *n {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		list = append(list, fmt.Sprintf("%s=%s", k, (*n)[k]))
	}

	return strings.Join(list, ",")
}

// Clone returns a copy of the tags
func (n *NodeTags) Clone() NodeTags {
	m := make(NodeTags, 0)
	for k, v := range *n {
		m[k] = v
	}

	return m
}

// Plugin represents a cloud provider plugin
type Plugin interface {
	// New returns an instance of a cloud provider
	New() (Provider, error)
}

// Provider is the cloud provider interface
type Provider interface {
	// GetNodeID returns our own node id
	GetNodeID() (NodeID, error)
	// DescribePools retrieves a list of compute pool
	DescribePools(NodeTags) ([]Pool, error)
	// GetNodeTags retrieves a list of node tags
	GetNodeTags(NodeID) (NodeTags, error)
	// GetNodeTag retrieves a specific node tag
	GetNodeTag(NodeID, string) (string, bool, error)
	// SetNodeTags is used to set a series of tags on a node
	SetNodeTags(NodeID, NodeTags) error
}

// providers is a map of registered providers
var providers = make(map[string]Plugin, 0)

// Register creates and returns a new cloud provider
func Register(name string, p Plugin) error {
	providers[name] = p
	return nil
}

// Get returns and instance of a provider
func Get(name string) (Provider, error) {
	p, found := providers[name]
	if !found {
		return nil, errors.New("provider not registered")
	}

	return p.New()
}
