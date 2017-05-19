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
	"encoding/json"
	"errors"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	log "github.com/Sirupsen/logrus"
	api "k8s.io/client-go/tools/clientcmd/api/v1"
)

const (
	// CompletedTagValue indicates the client has processed the token
	CompletedTagValue = "Success"
)

var (
	// ErrTimedOut means the operation has timed out
	ErrTimedOut = errors.New("operation timed out")
	// ErrConsumedToken means the registration token has already been consumed
	ErrConsumedToken = errors.New("token already consumed")
)

// Client is the client implementation
type Client struct {
	config Config
	client cloud.Provider
}

// New creates a new client
func New(cfg Config, provider cloud.Provider) (*Client, error) {
	if err := cfg.IsValid(); err != nil {
		return nil, err
	}

	return &Client{config: cfg, client: provider}, nil
}

// Start is responsible for retrieving the tokens from instance tags
func (c *Client) Start() (string, error) {
	var tmCh <-chan time.Time
	if c.config.Timeout > 0 {
		tmCh = time.After(c.config.Timeout)
	}
	intervalCh := time.NewTicker(1)
	firstTime := true

	for {
		select {
		case <-intervalCh.C:
			if firstTime {
				firstTime = false
				intervalCh.Stop()
				intervalCh = time.NewTicker(c.config.Interval)
			}
			token, found, err := c.consumeKubeletToken()
			if err == nil && found {
				return token, nil
			}
			if err == ErrConsumedToken {
				return "", ErrConsumedToken
			}
		case <-tmCh:
			return "", ErrTimedOut
		}
	}
}

// consumeKubeletToken is responsible for consuming the kubelet registration token
func (c *Client) consumeKubeletToken() (string, bool, error) {
	// step: get our instance id
	nodeID, err := c.client.GetNodeID()
	if err != nil {
		return "", false, err
	}
	// step: retrieve the tags for this node
	token, found, err := c.client.GetNodeTag(nodeID, c.config.TagName)
	if err != nil {
		log.WithFields(log.Fields{
			"id":    nodeID,
			"error": err.Error(),
		}).Error("unable to retrieve the node tags")

		return "", false, err
	}
	if !found {
		log.WithFields(log.Fields{
			"id":  nodeID,
			"tag": c.config.TagName,
		}).Debug("registration token not yet available")

		return "", false, nil
	}
	// step: check the token hasn't been consumed already
	if token == CompletedTagValue {
		return "", false, ErrConsumedToken
	}

	log.WithFields(log.Fields{
		"id":  nodeID,
		"tag": c.config.TagName,
	}).Info("found kubelet registration token")

	// step: update the tag to indicate we are done
	updateTags := cloud.NodeTags{
		c.config.TagName: CompletedTagValue,
	}

	if err := c.client.SetNodeTags(nodeID, updateTags); err != nil {
		log.WithFields(log.Fields{
			"id":    nodeID,
			"error": err.Error(),
		}).Error("unable to update the node tags")

		return "", false, err
	}

	return token, true, nil
}

// GenerateKubeconfig generates a bootstrap kubeconfig for us
func GenerateKubeconfig(token, master, caPath string) ([]byte, error) {
	cluster := api.Cluster{
		Server:               master,
		CertificateAuthority: caPath,
	}
	if caPath == "" {
		cluster.InsecureSkipTLSVerify = true
	}

	name := "bootstrap-context"
	clusterName := "cluster"

	cfg := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		AuthInfos: []api.NamedAuthInfo{
			{Name: name, AuthInfo: api.AuthInfo{Token: token}},
		},
		Clusters: []api.NamedCluster{
			{Name: clusterName, Cluster: cluster},
		},
		Contexts: []api.NamedContext{
			{
				Name:    name,
				Context: api.Context{Cluster: clusterName, AuthInfo: name},
			},
		},
		CurrentContext: name,
	}

	return json.MarshalIndent(&cfg, "", "  ")
}
