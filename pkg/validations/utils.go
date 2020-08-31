package validations

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/prometheus/client_golang/prometheus"
)

func getPromLabels(name, namespace, kind string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "name": name, "kind": kind}
}

func newGaugeVecMetric(name, help string, labelNames []string) (*prometheus.GaugeVec, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	m := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_%s", strings.ReplaceAll(operatorName, "-", "_"), name),
		Help: help,
	}, labelNames)

	return m, nil
}
