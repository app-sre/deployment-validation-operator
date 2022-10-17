package validations

import (
	"fmt"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"

	"github.com/prometheus/client_golang/prometheus"
)

func getPromLabels(namespaceUID, namespace, name, uid, kind string) prometheus.Labels {
	return prometheus.Labels{
		"namespace_uid": namespaceUID,
		"namespace":     namespace,
		"uid":           uid,
		"name":          name,
		"kind":          kind,
	}
}

func DeleteMetrics(namespace, namespaceUID, name, uid, kind string) {
	promLabels := getPromLabels(
		namespaceUID,
		namespace,
		uid,
		name,
		kind,
	)
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
