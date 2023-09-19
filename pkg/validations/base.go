package validations

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/prometheus/client_golang/prometheus"
)

// NewRequestFromObject converts a client.Object into
// a validation request. Note that the NamespaceUID of the
// request cannot be derived from the object and should
// be optionally be set after instantiation.
func NewRequestFromObject(obj client.Object) Request {
	return Request{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		UID:       string(obj.GetUID()),
	}
}

type Request struct {
	Kind         string
	Name         string
	Namespace    string
	NamespaceUID string
	UID          string
}

func (r *Request) ToPromLabels() prometheus.Labels {
	return prometheus.Labels{
		"kind":          r.Kind,
		"name":          r.Name,
		"namespace":     r.Namespace,
		"namespace_uid": r.NamespaceUID,
		"uid":           r.UID,
	}
}
