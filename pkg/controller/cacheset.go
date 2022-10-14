package controller

import (
	"sync"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cacheSet map[cacheSetKey]*validationCache
type cacheSetKey schema.GroupVersionKind

var once = sync.Once{}

func newCacheSetKey(obj client.Object) cacheSetKey {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return cacheSetKey(gvk)
}

func newCacheSet() *cacheSet {
	var cs *cacheSet
	once.Do(func() {
		cs = &cacheSet{}
	})
	// should panic if the newCacheSet() called more than once
	if cs == nil {
		panic("there should only be 1 cache set")
	}
	return cs
}

func (cs *cacheSet) objectAlreadyValidated(obj client.Object) bool {
	resourceType := newCacheSetKey(obj)
	validationCache, ok := (*cs)[resourceType]
	if !ok {
		return false
	}
	return validationCache.objectAlreadyValidated(obj)
}

func (cs *cacheSet) store(obj client.Object, status validations.ValidationStatus) {
	key := newCacheSetKey(obj)
	validationCache, ok := (*cs)[key]
	if !ok {
		(*cs)[key] = newValidationCache()
		validationCache = (*cs)[key]
	}
	validationCache.store(obj, status)
}
