package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	wrapperv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	wrapperv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	filePaths := os.Args[1:]
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	s := runtime.NewScheme()
	if err := wrapperv1.AddToScheme(s); err != nil {
		panic(fmt.Errorf("failed to add schema k8s-job-wrapper v1: %w", err))
	}
	if err := wrapperv2.AddToScheme(s); err != nil {
		panic(fmt.Errorf("failed to add schema k8s-job-wrapper v1: %w", err))
	}
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	for _, path := range filePaths {
		file, err := os.Open(path)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to read file: %v", err))
		}
		yamlreader := kyaml.NewYAMLReader(bufio.NewReader(file))
		for {
			b, err := yamlreader.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			raw, _, err := decoder.Decode(b, nil, nil)
			if err != nil {
				logger.Warn(fmt.Sprintf("failed to decode: %v", err))
			}
			_ = raw
		}
	}
}
