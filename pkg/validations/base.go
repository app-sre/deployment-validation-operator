package validations

import (
	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"golang.stackrox.io/kube-linter/pkg/lintcontext"
	"golang.stackrox.io/kube-linter/pkg/run"
)

var log = logf.Log.WithName("validations")

// RunValidations will run all the registered validations
func RunValidations(request reconcile.Request, obj client.Object, kind string, isDeleted bool) {
	log.V(2).Info("validation", "kind", kind)
	basePromLabels := getBasePromLabels(
		request.Namespace,
		request.Name,
		kind,
	)

	// If the object was deleted, then just delete the metrics and
	// do not run any validations
	fullPromLabels := engine.CheckRegistry
	if isDeleted {
		for _, check := range engine.CheckRegistry() {
			fullPromLabels = getFullPromLabels(basePromLabels, check)
			engine.DeleteMetrics(fullPromLabels)
		}
		return
	}

	// Only run checks against an object with no owners.  This should be
	// the object that controls the configuration
	if !utils.IsOwner(obj) {
		return
	}

	lintCtxs := []lintcontext.LintContext{}
	lintCtx := &lintContextImpl{}
	lintCtx.addObjects(lintcontext.Object{K8sObject: obj})
	lintCtxs = append(lintCtxs, lintCtx)
	result, err := run.Run(lintCtxs, engine.CheckRegistry(), engine.EnabledChecks())
	if err != nil {
		log.Error(err, "error running validations")
		return
	}

	// Clear labels from past run to ensure only results from this run
	// are reflected in the metrics
	engine.ClearMetrics(result.Reports, basePromLabels)

	for _, report := range result.Reports {
		logger := log.WithValues(
			"request.namespace", request.Namespace,
			"request.name", request.Name,
			"kind", kind,
			"validation", report.Check,
			"check_description", result.GetCheck(report.Check).Description,
			"check_remediation", report.Remediation,
			"check_failure_reason", report.Diagnostic.Message,
		)
		metric := engine.GetMetric(report.Check)
		if metric == nil {
			log.Error(nil, "no metric found for validation", report.Check)
		} else {
			fullPromLabels = getFullPromLabels(basePromLabels, check)
			metric.With(fullPromLabels).Set(1)
			logger.Info(report.Remediation)
		}
	}
}
