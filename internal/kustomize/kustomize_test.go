/*
Copyright 2025 pixiv Inc.

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

package kustomize_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixiv/k8s-job-wrapper/internal/kubectl"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	"github.com/stretchr/testify/assert"
)

type mockRunner struct {
	t             *testing.T
	resource      string
	patches       string
	kustomization string
}

func (m mockRunner) Run(_ context.Context, arg ...string) (string, error) {
	if !assert.Equal(m.t, 2, len(arg)) {
		return "", nil
	}
	dir := arg[1]
	m.assertFile(filepath.Join(dir, "resource.yaml"), m.resource)
	m.assertFile(filepath.Join(dir, "patches.yaml"), m.patches)
	m.assertFile(filepath.Join(dir, "kustomization.yaml"), m.kustomization)
	return "", nil
}

func (m mockRunner) assertFile(path, content string) {
	f, err := os.Open(path)
	if !assert.Nil(m.t, err) {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	b, err := io.ReadAll(f)
	if !assert.Nil(m.t, err) {
		return
	}
	assert.Equal(m.t, content, string(b))
}

const (
	group   = "batch"
	version = "v1"
	kind    = "Job"
	name    = "pi"
)

const resource = `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
      - name: pi
        image: perl:5.34.0
        command: ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4`
const patches = `- op: replace
  path: /spec/template/backoffLimit
  value: 1`
const kustomization = `resources:
- resource.yaml
patches:
- target:
    group: batch
    version: v1
    kind: Job
    name: pi
  path: patches.yaml`

const kustomized = `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  backoffLimit: 4
  template:
    backoffLimit: 1
    spec:
      containers:
      - command:
        - perl
        - -Mbignum=bpi
        - -wle
        - print bpi(2000)
        image: perl:5.34.0
        name: pi
      restartPolicy: Never
`

func newPatchRequest() *kustomize.PatchRequest {
	return &kustomize.PatchRequest{
		Group:    group,
		Version:  version,
		Kind:     kind,
		Name:     name,
		Resource: resource,
		Patches:  patches,
	}
}

func TestPatchRunner(t *testing.T) {
	t.Run("kustomizable", func(t *testing.T) {
		m := &mockRunner{
			t:             t,
			resource:      resource,
			patches:       patches,
			kustomization: kustomization,
		}
		_, _ = kustomize.NewPatchRunner(m).Patch(context.TODO(), newPatchRequest())
	})
	t.Run("kustomize", func(t *testing.T) {
		got, err := kustomize.NewPatchRunner(
			kubectl.NewCommand(os.Getenv("KUBECTL"))).Patch(context.TODO(),
			newPatchRequest(),
		)
		if !assert.Nil(t, err) {
			t.Logf("err=%v", err)
			return
		}
		assert.Equal(t, &kustomize.PatchResponse{
			Manifest: kustomized,
		}, got)
	})
}
