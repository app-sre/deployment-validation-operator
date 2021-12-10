package prometheus

import (
	"fmt"
	"net/http"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var PrometheusRegistry *prometheus.Registry

var log = logf.Log.WithName("prometheus")

func InitMetricsEndpoint(metricsPath string, metricsPort int32) {
	PrometheusRegistry = prometheus.NewRegistry()
	processCollector := prometheus.NewProcessCollector(
		prometheus.ProcessCollectorOpts{},
	)
	goCollector := prometheus.NewGoCollector()
	PrometheusRegistry.MustRegister(processCollector)
	PrometheusRegistry.MustRegister(goCollector)

	handler := promhttp.HandlerFor(PrometheusRegistry, promhttp.HandlerOpts{})
	http.Handle(fmt.Sprintf("/%s", metricsPath), handler)

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
		log.Error(err, "Prometheus metrics server stopped unexpectedly")
	}()
}
