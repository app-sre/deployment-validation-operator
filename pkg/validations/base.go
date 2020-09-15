package validations

import (
	"reflect"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("Validations")

type ValidationInterface interface {
	AppliesTo() map[string]struct{}
	Validate(reconcile.Request, interface{}, string, bool)
}

var validations []ValidationInterface

// AddValidation will add a validation to the list of validations
func AddValidation(v ValidationInterface) {
	validations = append(validations, v)
}

// RunValidations will run all the registered validations
func RunValidations(request reconcile.Request, obj interface{}, kind string, isDeleted bool) {
	log.V(2).Info("Validation", "kind", kind)
	for _, v := range validations {
		log.V(2).Info("checking", "kind", kind)
		if _, ok := v.AppliesTo()[kind]; ok {
			log.V(2).Info("running", "validation", reflect.TypeOf(v).String())
			v.Validate(request, obj, kind, isDeleted)
		}
	}
}
