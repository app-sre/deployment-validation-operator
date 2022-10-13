package utils

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Request struct {
	UID          string
	Name         string
	NamespaceUID string
	Namespace    string
}

func NewRequestFromObj(obj *unstructured.Unstructured) Request {
	return Request{
		UID:          string(obj.GetUID()),
		Name:         obj.GetName(),
		NamespaceUID: "",
		Namespace:    obj.GetNamespace(),
	}
}
