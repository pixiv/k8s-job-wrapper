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
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	pixivnetv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	"github.com/pixiv/k8s-job-wrapper/internal/v1tov2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

func main() {
	flag.Parse()
	filePaths := flag.Args()

	s := runtime.NewScheme()
	if err := pixivnetv1.AddToScheme(s); err != nil {
		panic(fmt.Errorf("failed to add schema k8s-job-wrapper v1: %w", err))
	}
	if err := pixivnetv2.AddToScheme(s); err != nil {
		panic(fmt.Errorf("failed to add schema k8s-job-wrapper v1: %w", err))
	}
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	new := make([]map[string]interface{}, 0)

	for _, path := range filePaths {
		file, err := os.Open(path)
		if err != nil {
			panic(fmt.Errorf("failed to read file: %w", err))
		}
		yamlreader := kyaml.NewYAMLReader(bufio.NewReader(file))
		for {
			b, err := yamlreader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			before, _, err := decoder.Decode(b, nil, nil)
			if err != nil {
				panic(fmt.Errorf("failed to decode: %w", err))
			}
			newObj, err := v1tov2.ToV2(before)
			if err != nil {
				panic(fmt.Errorf("failed to change pixiv.net/v1 to v2: %w", err))
			}

			for _, obj := range newObj {
				unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				if err != nil {
					panic(fmt.Errorf("failed to convert object to unstructured: %w", err))
				}
				delete(unstructured, "status")
				if metadata, ok := unstructured["metadata"].(map[string]interface{}); ok {
					delete(metadata, "creationTimestamp")
				}
				new = append(new, unstructured)
			}
		}
	}

	for i, obj := range new {
		b, err := yaml.Marshal(obj)
		if err != nil {
			panic(fmt.Errorf("failed to marshal object to yaml: %w", err))
		}
		fmt.Print(string(b))
		if i != len(new)-1 {
			fmt.Println("---")
		}
	}
}
