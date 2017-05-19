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

package client

import (
	"errors"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestClientTimeout(t *testing.T) {
	p := newFakeProviderSetup()
	c := newFakeConfig()
	c.Timeout = time.Duration(1) * time.Millisecond
	client, err := New(c, p)
	assert.NotNil(t, client)
	assert.NoError(t, err)
	tk, err := client.Start()
	assert.Empty(t, tk)
	assert.Error(t, err)
	assert.Equal(t, ErrTimedOut.Error(), err.Error())
}

func TestConsumeToken(t *testing.T) {
	p := newFakeProvider("test-node", cloud.NodeTags{
		"Name":      "test-id",
		"Role":      "compute",
		"KubeToken": "test-token",
	})
	c := newFakeConfig()
	c.Interval = time.Duration(50) * time.Millisecond
	client, err := New(c, p)
	assert.NotNil(t, client)
	assert.NoError(t, err)
	token, err := client.Start()
	assert.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestClientTokenConsumed(t *testing.T) {
	p := newFakeProvider("test-node", cloud.NodeTags{
		"Name":      "test-id",
		"Role":      "compute",
		"KubeToken": CompletedTagValue,
	})
	c, err := New(newFakeConfig(), p)
	assert.NotNil(t, c)
	assert.NoError(t, err)
	token, err := c.Start()
	assert.Empty(t, token)
	assert.Error(t, err)
	assert.Equal(t, ErrConsumedToken.Error(), err.Error())
}

func TestClientTokenWaiting(t *testing.T) {
	p := newFakeProviderSetup()
	c := newFakeConfig()
	c.Interval = time.Duration(10) * time.Millisecond
	c.Timeout = time.Duration(5) * time.Second
	client, err := New(c, p)
	assert.NotNil(t, client)
	assert.NoError(t, err)
	go func() {
		<-time.After(time.Duration(100) * time.Millisecond)
		p.SetNodeTags("test-node", cloud.NodeTags{
			c.TagName: "test",
		})
	}()
	token, err := client.Start()
	assert.NoError(t, err)
	assert.Equal(t, "test", token)
}

func TestConfigIsValid(t *testing.T) {
	cs := []struct {
		config Config
		Ok     bool
	}{
		{config: Config{}},
		{config: Config{Interval: time.Duration(10) * time.Second}},
		{
			config: Config{
				Interval: time.Duration(10) * time.Second,
				TagName:  "KubeToken",
			},
			Ok: true,
		},
	}
	for i, c := range cs {
		err := c.config.IsValid()
		if c.Ok && err != nil {
			t.Errorf("case %d should not have thrown error: %s", i, err)
			continue
		}
		if !c.Ok && err == nil {
			t.Errorf("case %d should have thrown an error", i)
			continue
		}
	}
}

func TestGenerateKubeConfig(t *testing.T) {
	cs := []struct {
		Token    string
		KubeAPI  string
		CAPath   string
		Expected string
	}{
		{
			Token:    "test-token",
			KubeAPI:  "https://127.0.0.1",
			Expected: "{\n  \"kind\": \"Config\",\n  \"apiVersion\": \"v1\",\n  \"preferences\": {},\n  \"clusters\": [\n    {\n      \"name\": \"cluster\",\n      \"cluster\": {\n        \"server\": \"https://127.0.0.1\",\n        \"insecure-skip-tls-verify\": true\n      }\n    }\n  ],\n  \"users\": [\n    {\n      \"name\": \"bootstrap-context\",\n      \"user\": {\n        \"token\": \"test-token\"\n      }\n    }\n  ],\n  \"contexts\": [\n    {\n      \"name\": \"bootstrap-context\",\n      \"context\": {\n        \"cluster\": \"cluster\",\n        \"user\": \"bootstrap-context\"\n      }\n    }\n  ],\n  \"current-context\": \"bootstrap-context\"\n}",
		},
		{
			Token:    "test-token",
			KubeAPI:  "https://127.0.0.1",
			CAPath:   "/etc/ssl/ca.pam",
			Expected: "{\n  \"kind\": \"Config\",\n  \"apiVersion\": \"v1\",\n  \"preferences\": {},\n  \"clusters\": [\n    {\n      \"name\": \"cluster\",\n      \"cluster\": {\n        \"server\": \"https://127.0.0.1\",\n        \"certificate-authority\": \"/etc/ssl/ca.pam\"\n      }\n    }\n  ],\n  \"users\": [\n    {\n      \"name\": \"bootstrap-context\",\n      \"user\": {\n        \"token\": \"test-token\"\n      }\n    }\n  ],\n  \"contexts\": [\n    {\n      \"name\": \"bootstrap-context\",\n      \"context\": {\n        \"cluster\": \"cluster\",\n        \"user\": \"bootstrap-context\"\n      }\n    }\n  ],\n  \"current-context\": \"bootstrap-context\"\n}",
		},
	}
	for i, c := range cs {
		config, err := GenerateKubeconfig(c.Token, c.KubeAPI, c.CAPath)
		if err != nil {
			t.Errorf("case %d should not have thrown error: %s", i, err)
			continue
		}
		assert.NotEmpty(t, config)
		assert.Equal(t, c.Expected, string(config))
	}
}

func TestNewClientBadConfig(t *testing.T) {
	client, err := New(Config{}, newFakeProviderSetup())
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNew(t *testing.T) {
	client, err := New(newFakeConfig(), newFakeProviderSetup())
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func newFakeConfig() Config {
	log.SetOutput(ioutil.Discard)
	return Config{
		Interval: time.Duration(5) * time.Second,
		TagName:  "KubeToken",
	}
}

func newFakeProviderSetup() cloud.Provider {
	return newFakeProvider("test-node", cloud.NodeTags{
		"Env":  "dev",
		"Role": "compute",
	})
}

type fakeProvider struct {
	sync.RWMutex
	tags   cloud.NodeTags
	nodeID cloud.NodeID
}

func newFakeProvider(nodeID cloud.NodeID, tags cloud.NodeTags) cloud.Provider {
	return &fakeProvider{
		tags:   tags,
		nodeID: nodeID,
	}
}

func (f *fakeProvider) GetNodeID() (cloud.NodeID, error) {
	return f.nodeID, nil
}

func (f *fakeProvider) DescribePools(cloud.NodeTags) ([]cloud.Pool, error) {
	return []cloud.Pool{}, errors.New("access denyed")
}

// GetNodeTags retrieves a list of node tags
func (f *fakeProvider) GetNodeTags(id cloud.NodeID) (cloud.NodeTags, error) {
	if f.nodeID == id {
		return f.tags, nil
	}

	return cloud.NodeTags{}, cloud.ErrInstanceNotFound
}

// GetNodeTag retrieves a specific node tag
func (f *fakeProvider) GetNodeTag(id cloud.NodeID, tag string) (string, bool, error) {
	f.RLock()
	defer f.RUnlock()
	if id != f.nodeID {
		return "", false, cloud.ErrInstanceNotFound
	}
	v, found := f.tags[tag]

	return v, found, nil
}

// SetNodeTags is used to set a series of tags on a node
func (f *fakeProvider) SetNodeTags(id cloud.NodeID, tags cloud.NodeTags) error {
	f.Lock()
	defer f.Unlock()
	if id != f.nodeID {
		return cloud.ErrInstanceNotFound
	}

	for k, v := range tags {
		f.tags[k] = v
	}

	return nil
}
