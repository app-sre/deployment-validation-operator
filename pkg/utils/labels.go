package utils

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

func GetLabels(object *unstructured.Unstructured) labels.Set {
	return labels.Set(object.GetLabels())
}
