package controller

import (
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type validationCache map[validationKey]validationResource

type validationKey struct {
	namespace, name, uid string
}

type resourceVersion string

type validationResource struct {
	version resourceVersion
	status  validations.ValidationStatus
}

func newResourceversionVal(str string) resourceVersion {
	return resourceVersion(str)
}

func newValidationKey(obj client.Object) validationKey {
	return validationKey{
		namespace: obj.GetNamespace(),
		name:      obj.GetName(),
		uid:       string(obj.GetUID()),
	}
}

func newValidationResource(rscVer resourceVersion, status validations.ValidationStatus) validationResource {
	return validationResource{
		version: rscVer,
		status:  status,
	}
}

func newValidationCache() *validationCache {
	return &validationCache{}
}

func (vc *validationCache) has(key validationKey) bool {
	_, ok := (*vc)[key]
	return ok
}

func (vc *validationCache) store(obj client.Object, status validations.ValidationStatus) {
	key := newValidationKey(obj)
	(*vc)[key] = newValidationResource(
		newResourceversionVal(obj.GetResourceVersion()),
		status,
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
	return vc.retrieveKey(key)
}

func (vc *validationCache) retrieveKey(key validationKey) (validationResource, bool) {
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
