package testing

import (
	"bytes"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func LoadUnstructuredFromFile(path string) (client.Object, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), 100)

	var rawObj runtime.RawExtension
	if err = decoder.Decode(&rawObj); err != nil {
		return nil, fmt.Errorf("decoding raw object: %w", err)
	}

	obj, _, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("decoding runtime object: %w", err)
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("converting to unstructured object: %w", err)
	}

	return &unstructured.Unstructured{Object: unstructuredMap}, nil
}
