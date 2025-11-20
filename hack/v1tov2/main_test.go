// Copyright 2[0-9]{3} pixiv Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestMain(t *testing.T) {
	var err error
	var cmd *exec.Cmd
	binaryPath := path.Join(t.TempDir(), "v1tov2")

	err = os.MkdirAll(filepath.Dir(binaryPath), 0755)
	if !assert.Nil(t, err, "failed to run mkdir") {
		return
	}

	cmd = exec.Command("go", "build", "-o", binaryPath, "main.go")
	err = cmd.Run()
	if !assert.Nil(t, err, "failed to run go build") {
		return
	}

	const manifestsInDir = "manifests/in"
	const manifestsOutDir = "manifests/out"

	entries, err := os.ReadDir(manifestsInDir)
	if err != nil {
		t.Fatalf("failed to read manifests/in directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		t.Run(filename, func(t *testing.T) {
			inputPath := filepath.Join(manifestsInDir, filename)
			outputPath := filepath.Join(manifestsOutDir, filename)

			expectedOutput, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read expected output file %s: %v", outputPath, err)
			}

			cmd := exec.Command(binaryPath, inputPath)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to run %s %s: %v\nstderr: %s", binaryPath, inputPath, err, stderr.String())
			}

			var got map[string]interface{}
			if err := yaml.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("failed to unmarshal output: %v", err)
			}
			var want map[string]interface{}
			if err := yaml.Unmarshal(expectedOutput, &want); err != nil {
				t.Fatalf("failed to unmarshal output: %v", err)
			}
			assert.Equal(t, want, got)
		})
	}
}
