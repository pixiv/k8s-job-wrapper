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

package kustomize

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pixiv/k8s-job-wrapper/internal/kubectl"
)

type Patcher interface {
	// 単一リソースにパッチを適用し, そのマニフェストを返す
	Patch(ctx context.Context, req *PatchRequest) (*PatchResponse, error)
}

var _ Patcher = &PatchRunner{}

type PatchRequest struct {
	Group    string
	Version  string
	Kind     string
	Name     string
	Resource string // オリジナルのマニフェスト
	Patches  string // パッチの内容
}

type PatchResponse struct {
	Manifest string
}

func NewPatchRunner(runner kubectl.Runner) *PatchRunner {
	return &PatchRunner{
		Runner: runner,
	}
}

type PatchRunner struct {
	kubectl.Runner
}

func (r PatchRunner) Patch(ctx context.Context, req *PatchRequest) (*PatchResponse, error) {
	dir, err := os.MkdirTemp("", "patchrunner")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	if err := r.generateKustomization(req, dir); err != nil {
		return nil, err
	}
	output, err := r.Run(ctx, "kustomize", dir)
	if err != nil {
		return nil, err
	}
	return &PatchResponse{
		Manifest: output,
	}, nil
}

func (r PatchRunner) generateKustomization(req *PatchRequest, dir string) error {
	const (
		resourcePath = "resource.yaml"
		patchesPath  = "patches.yaml"
	)
	if err := r.writeToFile(req.Resource, filepath.Join(dir, resourcePath)); err != nil {
		return err
	}
	if err := r.writeToFile(req.Patches, filepath.Join(dir, patchesPath)); err != nil {
		return err
	}
	content := fmt.Sprintf(`resources:
- %[1]s
patches:
- target:
    group: %[2]s
    version: %[3]s
    kind: %[4]s
    name: %[5]s
  path: %[6]s`,
		resourcePath,
		req.Group,
		req.Version,
		req.Kind,
		req.Name,
		patchesPath,
	)
	return r.writeToFile(content, filepath.Join(dir, "kustomization.yaml"))
}

func (PatchRunner) writeToFile(content, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	_, _ = fmt.Fprint(f, content)
	return nil
}
