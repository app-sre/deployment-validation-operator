package validations

import (
	"fmt"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"

	"github.com/prometheus/client_golang/prometheus"
)

func getPromLabels(namespaceUID, namespace, uid, name, kind string) prometheus.Labels {
	return prometheus.Labels{
		"namespace_uid": namespaceUID,
		"namespace":     namespace,
		"uid":           uid,
		"name":          name,
		"kind":          kind,
	}
}

func DeleteMetrics(promLabels prometheus.Labels) {
	engine.DeleteMetrics(promLabels)
}

func newGaugeVecMetric(
	name, help string, labelNames []string, constLabels prometheus.Labels,
) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        fmt.Sprintf("%s_%s", strings.ReplaceAll(config.OperatorName, "-", "_"), name),
			Help:        help,
			ConstLabels: constLabels,
		},
		labelNames,
	)
}
