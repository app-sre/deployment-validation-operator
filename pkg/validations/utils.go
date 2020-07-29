package validations

import (
	"github.com/prometheus/client_golang/prometheus"
)

func getPromLabels(name, namespace, kind string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "name": name, "kind": kind}
}
