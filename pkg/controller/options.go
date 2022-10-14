package controller

import (
	"regexp"

	"github.com/go-logr/logr"
)

type WithIgnorePattern struct{ Pattern *regexp.Regexp }

func (w WithIgnorePattern) ConfigureWatchNamespacesCache(c *watchNamespacesCacheConfig) {
	c.IgnorePattern = w.Pattern
}

type WithListLimit int

func (w WithListLimit) ConfigureGenericReconcilier(c *GenericReconcilierConfig) {
	c.ListLimit = int64(w)
}

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureGenericReconcilier(c *GenericReconcilierConfig) {
	c.Log = w.Log
}

type WithNamespaceCache struct{ Cache NamespaceCache }

func (w WithNamespaceCache) ConfigureGenericReconcilier(c *GenericReconcilierConfig) {
	c.NamespaceCache = w.Cache
}
