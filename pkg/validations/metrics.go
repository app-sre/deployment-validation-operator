package validations

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	metricRequestsLimits = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dv_requests_limits",
		Help: "resource does not have requests or limits.",
	}, []string{"namespace", "name", "kind"})

	metricReplicas = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dv_replicas",
		Help: "resource has less than 3 replicas.",
	}, []string{"namespace", "name", "kind"})
)

func init() {
	metrics.Registry.MustRegister(metricRequestsLimits)
	metrics.Registry.MustRegister(metricReplicas)
}
