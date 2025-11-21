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
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

const (
	annotationPrefix = "v1tov2.pixiv.net/"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `v1tov2 is a tool for converting pixiv.net/v1 resources to pixiv.net/v2 resources.
		
Usage:

	v1tov2 <filename> [another filenames...]

`)
	}
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
	newObjs := make([]map[string]any, 0)

	for _, path := range filePaths {
		file, err := os.Open(path)
		if err != nil {
			klog.Fatalf("failed to open file: %v\n", err)
		}
		defer file.Close()
		yamlreader := kyaml.NewYAMLReader(bufio.NewReader(file))
		for {
			b, err := yamlreader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			before, _, err := decoder.Decode(b, nil, nil)
			if err != nil {
				klog.Fatalf("failed to decode manifest: %v\n", err)
			}
			newObj, changed, err := v1tov2.ToV2(before)
			if err != nil {
				klog.Fatalf("failed to change pixiv.net/v1 to v2: %v\n", err)
			}

			for _, obj := range newObj {
				unstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				if err != nil {
					klog.Fatalf("failed to convert object to unstructured: %v", err)
				}
				delete(unstructured, "status")
				if metadata, ok := unstructured["metadata"].(map[string]interface{}); ok {
					delete(metadata, "creationTimestamp")
					if changed {
						if metadata["annotations"] == nil {
							metadata["annotations"] = make(map[string]string)
						}
						annotations := metadata["annotations"].(map[string]string)
						annotations[annotationPrefix+"kind"] = before.GetObjectKind().GroupVersionKind().Kind
						annotations[annotationPrefix+"name"] = metadata["name"].(string) // the name of the 'before' resource is same as 'after' resource
					}
				}
				newObjs = append(newObjs, unstructured)
			}
		}
	}

	for i, obj := range newObjs {
		b, err := yaml.Marshal(obj)
		if err != nil {
			klog.Fatalf("failed to marshal object to yaml: %v\n", err)
		}
		fmt.Print(string(b))
		if i != len(newObjs)-1 {
			fmt.Println("---")
		}
	}
}
