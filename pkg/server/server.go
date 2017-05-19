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
	"fmt"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	log "github.com/Sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Server is the service component
type Server struct {
	cm     cloud.Provider
	config Config
	kube   *kubernetes.Clientset
	tokens TokensProvider
}

// New creates a new kubelet registration service
func New(cfg Config, p cloud.Provider, t TokensProvider) (*Server, error) {
	log.WithFields(log.Fields{
		"filters":  cfg.Filters.String(),
		"tag-name": cfg.TagName,
		"ttl":      cfg.TokenTTL,
	}).Infof("starting the kubernetes token service")

	// step: create a kube client
	kube, err := getKubeClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Server{
		cm:     p,
		config: cfg,
		kube:   kube,
		tokens: t,
	}, nil
}

// Start engages the kubelet registration service
func (s *Server) Start() error {
	firstTime := true
	checkCh := time.NewTicker(1)
	for {
		select {
		case <-checkCh.C:
			if firstTime {
				checkCh.Stop()
				checkCh = time.NewTicker(s.config.ReconcileInterval)
			}
			s.reconcileComputeNodes()
		}
	}
}

// reconcileComputeNodes is responsible for finding new instance and generating
// registration tokens for them
func (s *Server) reconcileComputeNodes() error {
	pools, err := s.cm.DescribePools(s.config.Filters)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("failed to get list of node pools")

		return nil
	}
	log.Debugf("found %d node pools tagged", len(pools))

	usages := []string{"authentication", "signing"}
	nodesCh := make(chan cloud.NodeID, 10)
	go func() {
		for _, pool := range pools {
			for _, node := range pool.Nodes {
				_, found, err := s.cm.GetNodeTag(node, s.config.TagName)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
						"node":  node,
						"pool":  pool.Name,
					}).Error("failed to get instance tag")

					continue
				}
				// check: if the tags if found move on
				if found {
					log.WithFields(log.Fields{
						"node": node,
						"pool": pool.Name,
					}).Debug("skipping node as token already set")

					continue
				}
				nodesCh <- node
			}
		}
		close(nodesCh)
	}()

	for node := range nodesCh {
		err := func(n cloud.NodeID) error {
			token, err := s.tokens.Create(s.kube, n, s.config.TokenTTL, usages, s.config.TokenNamespace)
			if err != nil {
				return fmt.Errorf("failed to create token, error: %s", err)
			}
			updateTags := cloud.NodeTags{s.config.TagName: token}

			if err := s.cm.SetNodeTags(n, updateTags); err != nil {
				if err = s.tokens.Delete(s.kube, token, s.config.TokenNamespace); err != nil {
					return fmt.Errorf("failed to delete the create token on failure to update tags, error: %s", err)
				}
				return fmt.Errorf("failed to update tags, error: %s", err)
			}
			return nil
		}(node)
		if err != nil {
			log.WithFields(log.Fields{
				"node":  node,
				"error": err.Error(),
			}).Error("failed to create registration token")

			continue
		}

		log.WithFields(log.Fields{
			"node":    node,
			"expires": time.Now().Add(s.config.TokenTTL).Format(time.RFC1123Z),
		}).Info("successfully generate token for node")
	}

	return nil
}

// getKubeClient is responsible for creating a kubernetes API client for us
func getKubeClient(c Config) (*kubernetes.Clientset, error) {
	var err error
	var config *rest.Config
	if c.MasterAPI != "" && c.KubeToken != "" {
		config = &rest.Config{
			Host:        c.MasterAPI,
			BearerToken: c.KubeToken,
			Insecure:    true,
		}
	} else if c.KubeConfig != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: c.KubeConfig},
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}
