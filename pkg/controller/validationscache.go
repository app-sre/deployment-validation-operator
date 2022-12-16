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

// newValidationKey returns an instance of validationKey struct
// populated with data extrated from the given object
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
	uid     string
	outcome validations.ValidationOutcome
}

// newValidationResource returns a new empty instance of validationResource struct
func newValidationResource(
	rscVer resourceVersion,
	uid string,
	outcome validations.ValidationOutcome,
) validationResource {
	return validationResource{
		uid:     uid,
		version: rscVer,
		outcome: outcome,
	}
}

type validationCache map[validationKey]validationResource

// newValidationCache returns a new empty instance of validationCache struct
func newValidationCache() *validationCache {
	return &validationCache{}
}

// has returns a boolean if the given key exist in the instance
func (vc *validationCache) has(key validationKey) bool {
	_, exists := (*vc)[key]
	return exists
}

// store creates a key-value pair on the current instance
// it uses given object to create a validationKey and a validationResource structs
// constraint: if a key already exists, it will overwrite the value
func (vc *validationCache) store(obj client.Object, outcome validations.ValidationOutcome) {
	key := newValidationKey(obj)
	(*vc)[key] = newValidationResource(
		newResourceversionVal(obj.GetResourceVersion()),
		string(obj.GetUID()),
		outcome,
	)
}

// drain overwrites the current instance with an empty one
// all keys and values will be lost
func (vc *validationCache) drain() {
	*vc = validationCache{}
}

// remove deletes a key, and its value, from the instance
// it uses given object to search for the validationKey
func (vc *validationCache) remove(obj client.Object) {
	key := newValidationKey(obj)
	vc.removeKey(key)
}

// removeKey deletes a key, and its value, from the instance
func (vc *validationCache) removeKey(key validationKey) {
	delete(*vc, key)
}

// retrieve returns the value, if exists, within a key
// it uses given object to search for the validationKey
// it returns a second parameter to check if the key existed in the instance
func (vc *validationCache) retrieve(obj client.Object) (validationResource, bool) {
	key := newValidationKey(obj)
	val, exists := (*vc)[key]
	return val, exists
}

// objectAlreadyValidated returns if the given object has been validated
func (vc *validationCache) objectAlreadyValidated(obj client.Object) bool {
	validationOutcome, exists := vc.retrieve(obj)
	if exists {
		storedResourceVersion := string(validationOutcome.version)
		currentResourceVersion := obj.GetResourceVersion()

		if storedResourceVersion != currentResourceVersion {
			vc.remove(obj)
			return false
		}
	}

	return exists
}
