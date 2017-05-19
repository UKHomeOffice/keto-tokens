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

// Note: most if not all of this was copied from kubeadm
// https://github.com/kubernetes/kubernetes/tree/master/cmd/kubeadm/app/phases/token

// Notes: https://github.com/kubernetes/kubernetes/pull/41281

import (
	"errors"
	"fmt"
	"time"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	bootstrapapi "k8s.io/kubernetes/pkg/bootstrap/api"
)

// kubeTokensProvider reuses the kubeadm code for tokens
type kubeTokensProvider struct{}

// NewTokenProvider creates and returns a tokensProvider
func NewTokenProvider() (TokensProvider, error) {
	return &kubeTokensProvider{}, nil
}

const (
	tokenNamespace = "kube-system"
)

// Create generates a token for the instance
func (c *kubeTokensProvider) Create(client *kubernetes.Clientset, id cloud.NodeID, ttl time.Duration, usages []string, namespace string) (string, error) {
	newToken, err := generateToken()
	if err != nil {
		return "", err
	}
	tokenID, tokenSecret, err := parseToken(newToken)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s%s", bootstrapapi.BootstrapTokenSecretPrefix, tokenID)
	for i := 0; i < 5; i++ {
		if found, err := c.hasToken(client, name, namespace); err != nil {
			continue
		} else if found {
			return newToken, nil
		}

		// step: add the secret to the namespace
		secret := &v1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name: name,
			},
			Type: v1.SecretType(bootstrapapi.SecretTypeBootstrapToken),
			Data: encodeTokenSecretData(tokenID, tokenSecret, usages, ttl),
		}

		if _, err := client.Secrets(namespace).Create(secret); err == nil {
			return newToken, nil
		}
	}

	return "", errors.New("failed to added the token after multiple attempts")
}

// Delete remove a token from the token namespace
func (c *kubeTokensProvider) Delete(client *kubernetes.Clientset, token, namespace string) error {
	// step: parse token to grab the ID
	tokenID, _, err := parseToken(token)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s%s", bootstrapapi.BootstrapTokenSecretPrefix, tokenID)
	if found, err := c.hasToken(client, name, namespace); err != nil {
		return err
	} else if !found {
		return nil
	}

	return client.Secrets(tokenNamespace).Delete(name, &v1.DeleteOptions{})
}

// hasToken checks if a secret exists in the namespace
func (c *kubeTokensProvider) hasToken(client *kubernetes.Clientset, name, namespace string) (bool, error) {
	list, err := client.Secrets(namespace).List(v1.ListOptions{})
	if err != nil {
		return false, err
	}
	for _, x := range list.Items {
		if x.Name == name {
			return true, nil
		}
	}

	return false, nil
}
