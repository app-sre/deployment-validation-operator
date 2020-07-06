package validations

import (
	"reflect"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/prometheus/client_golang/prometheus"
)

var log = logf.Log.WithName("Validations")

type ValidationInterface interface {
	AppliesTo() map[string]struct{}
	Validate(reconcile.Request, string, interface{}, bool)
	ValidateWithClient(client.Client)
}

var validations []ValidationInterface

func getPromLabels(name, namespace, kind string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "name": name, "kind": kind}
}

// AddValidation will add a validation to the list of validations
func AddValidation(v ValidationInterface) {
	validations = append(validations, v)
}

// RunValidations will run all the registered validations
func RunValidations(request reconcile.Request, obj interface{}, isDeleted bool) {
	kind := reflect.TypeOf(obj).String()
	log.Info("Validation", "kind", kind)
	kind = strings.SplitN(kind, ".", 2)[1]
	for _, v := range validations {
		log.Info("checking", "kind", kind)
		if _, ok := v.AppliesTo()[kind]; ok {
			log.Info("running", "validation", reflect.TypeOf(v).String())
			v.Validate(request, kind, reflect.ValueOf(obj).Elem().Interface(), isDeleted)
		}
	}
}

//func RunValidationsWithClient(kubeClient client.Client) {
func RunValidationsWithClient(kubeClient client.Client) {
	for _, v := range validations {
		v.ValidateWithClient(kubeClient)
	}
}
