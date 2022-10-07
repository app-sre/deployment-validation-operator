package utils

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Request struct {
	NameUID      string
	Name         string
	NamespaceUID string
	Namespace    string
}

func NewRequestFromObj(obj *unstructured.Unstructured) Request {
	return Request{
		NameUID:      string(obj.GetUID()),
		Name:         obj.GetName(),
		NamespaceUID: "",
		Namespace:    obj.GetNamespace(),
	}
}
