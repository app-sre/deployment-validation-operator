package validations

import (
	"fmt"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"

	"github.com/prometheus/client_golang/prometheus"
	kubelinterconfig "golang.stackrox.io/kube-linter/pkg/config"
)

func getBasePromLabels(namespace, name, kind string) prometheus.Labels {
	return prometheus.Labels{
		"namespace": namespace,
		"name": name,
		"kind": kind,
	}
}

func getFullPromLabels(basePromLabels prometheus.Labels, check kubelinterconfig.Check,
	) prometheus.Labels {
	fullPromLabels := prometheus.Labels{
		"check_description": check.Description,
		"check_remediation": check.Remediation,
	}
	for k, v := range basePromLabels {
		fullPromLabels[k] = v
	}
	return fullPromLabels
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
