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
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakePlugin struct{}
type fakeProvider struct {
	Provider
}

func (f *fakePlugin) New() (Provider, error) {
	return &fakeProvider{}, nil
}

func TestNodeString(t *testing.T) {
	cs := []struct {
		Tags     NodeTags
		Expected string
	}{
		{Tags: NodeTags{}},
		{Tags: NodeTags{"Role": "compute"}, Expected: "Role=compute"},
		{Tags: NodeTags{"Role": "compute", "Env": "dev"}, Expected: "Env=dev,Role=compute"},
	}
	for i, c := range cs {
		s := c.Tags.String()
		assert.Equal(t, c.Expected, s, "case %d, expected: %s, got: %s", i, c.Expected, s)
	}
}

func TestNodeClone(t *testing.T) {
	p := NodeTags{"Role": "compute"}
	n := p.Clone()
	assert.Equal(t, p, n)
	n["test"] = "test"
	assert.Empty(t, p["test"])
	assert.NotEmpty(t, n["test"])
}

func TestRegister(t *testing.T) {
	err := Register("test", &fakePlugin{})
	assert.NoError(t, err)
	assert.Contains(t, providers, "test")
}

func TestGet(t *testing.T) {
	v := &fakePlugin{}
	err := Register("test", v)
	assert.NoError(t, err)
	assert.Contains(t, providers, "test")
	p, err := Get("test")
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestGetError(t *testing.T) {
	p, err := Get("not_there")
	assert.Error(t, err)
	assert.Nil(t, p)
}
