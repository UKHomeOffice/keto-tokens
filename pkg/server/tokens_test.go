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

	"k8s.io/client-go/kubernetes"

	"github.com/UKHomeOffice/keto-tokens/pkg/cloud"
)

type fakeTokenProvider struct {
	tokens map[string]cloud.NodeID
}

func newFakeTokenProvider() TokensProvider {
	return &fakeTokenProvider{tokens: make(map[string]cloud.NodeID, 0)}
}

func (f *fakeTokenProvider) Create(client *kubernetes.Clientset, id cloud.NodeID,
	ttl time.Duration, usages []string, namespace string) (string, error) {
	newTokens, err := generateToken()
	if err != nil {
		return "", err
	}
	f.tokens[newTokens] = id

	return newTokens, nil
}

func (f *fakeTokenProvider) Delete(client *kubernetes.Clientset, token, namespace string) error {
	delete(f.tokens, token)
	return nil
}
