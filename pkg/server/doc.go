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
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	"k8s.io/client-go/kubernetes"
)

// server holds the code for generating the registration tokens for the compute worker nodes

// Config is the configuration for the server
type Config struct {
	// MasterAPI is the URL for the Kubernetes API
	MasterAPI string
	// KubeToken is a user defined kube token
	KubeToken string
	// KubeConfig the path to a kubeconfig
	KubeConfig string
	// TokenTTL is the default token ttl
	TokenTTL time.Duration
	// TokenNamespace is the namespace to registration token
	TokenNamespace string
	// Filters is a collection of filter used to identify the compute nodes
	Filters cloud.NodeTags
	// TagName is the name of the registration token tag
	TagName string
	// ReconcileInterval is the checking interval for new instances
	ReconcileInterval time.Duration
	// AcquireLock indicates we must acquire the lock in kubernetes
	AcquireLock bool
}

// TokensProvider implements the interactions with the kubeapi and tokens
type TokensProvider interface {
	// Create genenates a registration token
	Create(*kubernetes.Clientset, cloud.NodeID, time.Duration, []string, string) (string, error)
	// Delete remove a token
	Delete(*kubernetes.Clientset, string, string) error
}
