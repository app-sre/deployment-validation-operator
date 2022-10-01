package controller

import (
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type validationKey struct {
	group, version, kind, namespace, name string
}

type resourceVersion string

func newResourceversionVal(str string) resourceVersion {
	return resourceVersion(str)
}

func newValidationKey(obj client.Object) validationKey {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return validationKey{
		group:     gvk.Group,
		version:   gvk.Version,
		kind:      gvk.Kind,
		namespace: obj.GetNamespace(),
		name:      obj.GetName(),
	}
}

type validationResource struct {
	version resourceVersion
	outcome validations.ValidationOutcome
}

func newValidationResource(rscVer resourceVersion, outcome validations.ValidationOutcome) validationResource {
	return validationResource{
		version: rscVer,
		outcome: outcome,
	}
}

type validationCache map[validationKey]validationResource

func newValidationCache() *validationCache {
	return &validationCache{}
}

func (vc *validationCache) has(key validationKey) bool {
	_, ok := (*vc)[key]
	return ok
}

func (vc *validationCache) store(obj client.Object, outcome validations.ValidationOutcome) {
	key := newValidationKey(obj)
	(*vc)[key] = newValidationResource(
		newResourceversionVal(obj.GetResourceVersion()),
		outcome,
	)
}

func (vc *validationCache) drain() {
	*vc = validationCache{}
}

func (vc *validationCache) remove(obj client.Object) {
	key := newValidationKey(obj)
	vc.removeKey(key)
}

func (vc *validationCache) removeKey(key validationKey) {
	delete(*vc, key)
}

func (vc *validationCache) retrieve(obj client.Object) (validationResource, bool) {
	key := newValidationKey(obj)
	val, ok := (*vc)[key]
	return val, ok
}

func (vc *validationCache) objectAlreadyValidated(obj client.Object) bool {
	validationOutcome, ok := vc.retrieve(obj)
	storedResourceVersion := validationOutcome.version
	if !ok {
		return false
	}
	currentResourceVersion := obj.GetResourceVersion()
	if string(storedResourceVersion) != currentResourceVersion {
		vc.remove(obj)
		return false
	}
	return true
}
