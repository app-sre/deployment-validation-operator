package validations

import (
	"fmt"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"

	"github.com/prometheus/client_golang/prometheus"
)

func getPromLabels(name, namespace, kind string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "name": name, "kind": kind}
}

func newGaugeVecMetric(name, help string, labelNames []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", strings.ReplaceAll(config.OperatorName, "-", "_"), name),
		Help: help,
	}, labelNames)
}
