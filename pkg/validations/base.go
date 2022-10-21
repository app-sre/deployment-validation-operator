package validations

import (
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/app-sre/deployment-validation-operator/pkg/utils"

	"github.com/prometheus/client_golang/prometheus"
	"golang.stackrox.io/kube-linter/pkg/lintcontext"
	"golang.stackrox.io/kube-linter/pkg/run"
)

var log = logf.Log.WithName("validations")

type ValidationOutcome string

var (
	ObjectNeedsImprovement  ValidationOutcome = "object needs improvement"
	ObjectValid             ValidationOutcome = "object valid"
	ObjectValidationIgnored ValidationOutcome = "object validation ignored"
)

// RunValidations will run all the registered validations
func RunValidations(request Request, obj client.Object) (ValidationOutcome, error) {
	log.V(2).Info("validation", "kind", request.Kind)

	promLabels := request.ToPromLabels()

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
			"kind", request.Kind,
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

// NewRequestFromObject converts a client.Object into
// a validation request. Note that the NamespaceUID of the
// request cannot be derived from the object and should
// be optionally be set after instantiation.
func NewRequestFromObject(obj client.Object) Request {
	return Request{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		UID:       string(obj.GetUID()),
	}
}

type Request struct {
	Kind         string
	Name         string
	Namespace    string
	NamespaceUID string
	UID          string
}

func (r *Request) ToPromLabels() prometheus.Labels {
	return prometheus.Labels{
		"kind":          r.Kind,
		"name":          r.Name,
		"namespace":     r.Namespace,
		"namespace_uid": r.NamespaceUID,
		"uid":           r.UID,
	}
}
