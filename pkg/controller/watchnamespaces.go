package controller

import (
	"context"
	"fmt"
	"os"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type namespace struct {
	uid, name string
}

type watchNamespacesCache struct {
	namespaces    *[]namespace
	ignorePattern *regexp.Regexp
}

func newWatchNamespacesCache() *watchNamespacesCache {
	ignorePatternStr := os.Getenv(EnvNamespaceIgnorePattern)
	var nsIgnoreRegex *regexp.Regexp
	if ignorePatternStr != "" {
		nsIgnoreRegex = regexp.MustCompile(ignorePatternStr)
	}
	return &watchNamespacesCache{
		ignorePattern: nsIgnoreRegex,
	}
}

func (nsc *watchNamespacesCache) setCache(namespaces *[]namespace) {
	nsc.namespaces = namespaces
}

func (nsc *watchNamespacesCache) getNamespaceUID(namespace string) string {
	for _, ns := range *nsc.namespaces {
		if ns.name == namespace {
			return ns.uid
		}
	}
	return ""
}

func (nsc *watchNamespacesCache) resetCache() {
	nsc.setCache(nil)
}

func (nsc *watchNamespacesCache) getWatchNamespaces(ctx context.Context, c client.Client) (*[]namespace, error) {
	if nsc.namespaces != nil {
		return nsc.namespaces, nil
	}
	namespaceList := corev1.NamespaceList{}
	if err := c.List(ctx, &namespaceList); err != nil {
		return nil, fmt.Errorf("listing %s: %w", namespaceList.GroupVersionKind().String(), err)
	}
	watchNamespaces := []namespace{}
	for _, ns := range namespaceList.Items {
		name := ns.GetName()
		if nsc.ignorePattern != nil && nsc.ignorePattern.Match([]byte(name)) {
			continue
		}
		watchNamespaces = append(watchNamespaces, namespace{uid: string(ns.GetUID()), name: name})
	}
	nsc.setCache(&watchNamespaces)
	return nsc.namespaces, nil
}
