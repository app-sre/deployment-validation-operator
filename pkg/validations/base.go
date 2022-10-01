package validations

import (
	"fmt"
	"reflect"

	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"golang.stackrox.io/kube-linter/pkg/lintcontext"
	"golang.stackrox.io/kube-linter/pkg/run"
)

var log = logf.Log.WithName("validations")

func DeleteMetrics(namespace, name, kind string) {
	promLabels := getPromLabels(
		namespace,
		name,
		kind,
	)
	engine.DeleteMetrics(promLabels)
}

type ValidationOutcome string

var (
	ObjectNeedsImprovement  ValidationOutcome = "object needs improvement"
	ObjectValid             ValidationOutcome = "object valid"
	ObjectValidationIgnored ValidationOutcome = "object validation ignored"
)

// RunValidations will run all the registered validations
func RunValidations(request reconcile.Request, obj client.Object) (ValidationOutcome, error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	log.V(2).Info("validation", "kind", kind)

	promLabels := getPromLabels(request.Namespace, request.Name, kind)

	// Only run checks against an object with no owners.  This should be
	// the object that controls the configuration
	if !utils.IsOwner(obj) {
		return ObjectValidationIgnored, nil
	}

	// If controller has no replicas clear existing metrics and
	// do not run any validations

	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if spec.IsValid() {
		replicas := spec.FieldByName("Replicas")
		if replicas.IsValid() {
			numReplicas, ok := replicas.Interface().(*int32)

			// clear labels if we fail to get a value for numReplicas, or if value is <= 0
			if !ok || numReplicas == nil || *numReplicas <= 0 {
				engine.DeleteMetrics(promLabels)
				return ObjectValidationIgnored, nil
			}
		}
	}

	lintCtxs := []lintcontext.LintContext{}
	lintCtx := &lintContextImpl{}
	lintCtx.addObjects(lintcontext.Object{K8sObject: obj})
	lintCtxs = append(lintCtxs, lintCtx)
	result, err := run.Run(lintCtxs, engine.CheckRegistry(), engine.EnabledChecks())
	if err != nil {
		log.Error(err, "error running validations")
		return "", fmt.Errorf("error running validations: %v", err)
	}

	// Clear labels from past run to ensure only results from this run
	// are reflected in the metrics
	engine.ClearMetrics(result.Reports, promLabels)

	outcome := ObjectValid
	for _, report := range result.Reports {
		check, err := engine.GetCheckByName(report.Check)
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to get check '%s' by name", report.Check))
			return "", fmt.Errorf("error running validations: %v", err)
		}
		logger := log.WithValues(
			"request.namespace", request.Namespace,
			"request.name", request.Name,
			"kind", kind,
			"validation", report.Check,
			"check_description", check.Description,
			"check_remediation", report.Remediation,
			"check_failure_reason", report.Diagnostic.Message,
		)
		metric := engine.GetMetric(report.Check)
		if metric == nil {
			log.Error(nil, "no metric found for validation", report.Check)
		} else {
			metric.With(promLabels).Set(1)
			logger.Info(report.Remediation)
			outcome = ObjectNeedsImprovement
		}
	}
	return outcome, nil
}
