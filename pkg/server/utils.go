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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"

	bootstrapapi "k8s.io/kubernetes/pkg/bootstrap/api"
)

const (
	tokenIDBytes     = 3
	tokenSecretBytes = 8
)

var (
	tokenIDRegexpString = "^([a-z0-9]{6})$"
	tokenIDRegexp       = regexp.MustCompile(tokenIDRegexpString)
	tokenRegexpString   = "^([a-z0-9]{6})\\.([a-z0-9]{16})$"
	tokenRegexp         = regexp.MustCompile(tokenRegexpString)
)

// randBytes generates some random bytes
func randBytes(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// generateToken generates a new token with a token ID that is valid as a
func generateToken() (string, error) {
	tokenID, err := randBytes(tokenIDBytes)
	if err != nil {
		return "", err
	}
	tokenSecret, err := randBytes(tokenSecretBytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", tokenID, tokenSecret), nil
}

// parseTokenID tries and parse a valid token ID from a string.
func parseTokenID(s string) error {
	if !tokenIDRegexp.MatchString(s) {
		return fmt.Errorf("token ID [%q] was not of form [%q]", s, tokenIDRegexpString)
	}
	return nil
}

// parseToken tries and parse a valid token from a string.
func parseToken(s string) (string, string, error) {
	split := tokenRegexp.FindStringSubmatch(s)
	if len(split) != 3 {
		return "", "", fmt.Errorf("token [%q] was not of form [%q]", s, tokenRegexpString)
	}
	return split[1], split[2], nil
}

// encodeTokenSecretData takes the token discovery object and an optional duration and returns the .Data for the Secret
func encodeTokenSecretData(token, secret string, usages []string, ttl time.Duration) map[string][]byte {
	data := map[string][]byte{
		bootstrapapi.BootstrapTokenIDKey:     []byte(token),
		bootstrapapi.BootstrapTokenSecretKey: []byte(secret),
	}
	if ttl > 0 {
		expire := time.Now().Add(ttl).Format(time.RFC3339)
		data[bootstrapapi.BootstrapTokenExpirationKey] = []byte(expire)
	}
	for _, usage := range usages {
		data[bootstrapapi.BootstrapTokenUsagePrefix+usage] = []byte("true")
	}

	return data
}
