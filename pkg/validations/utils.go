package validations

import (
	"fmt"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"

	"github.com/prometheus/client_golang/prometheus"
)

func getPromLabels(namespace, name, kind string) prometheus.Labels {
	return prometheus.Labels{
		"namespace": namespace,
		"name": name,
		"kind": kind,
	}
}

func newGaugeVecMetric(
	name, help string, labelNames []string, constLabels prometheus.Labels,
	) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_%s", strings.ReplaceAll(config.OperatorName, "-", "_"), name),
			Help: help,
			ConstLabels: constLabels,
		},
		labelNames,
	)
}
