package controller

import (
	"context"
	"fmt"
	"os"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO : Evaluate changing uid type: string -> types.UID
// This data is populated with types.UID from corev1/ObjectMeta
// No access to this property out of the method getNamespaceUID
type namespace struct {
	uid, name string
}

type watchNamespacesCache struct {
	namespaces    *[]namespace
	ignorePattern *regexp.Regexp
}

// newWatchNamespacesCache returns a new watchNamespacesCache instance
// if EnvNamespaceIgnorePattern Env variable is set, ignorePattern field
// is populated with the information in the variable
// sidenote: regexp will throw a Panic in case data cannot be parsed
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

// getFormattedNamespaces returns a list of namespaces filtering through given list
// and formatting them as namespace structs
func getFormattedNamespaces(list corev1.NamespaceList, ignorePattern *regexp.Regexp) (fns []namespace) {
	for _, ns := range list.Items {
		name := ns.GetName()
		if ignorePattern == nil || !ignorePattern.Match([]byte(name)) {
			fns = append(fns, namespace{uid: string(ns.GetUID()), name: name})
		}
	}
	return
}

// setCache is a setter for the namespaces field
func (nsc *watchNamespacesCache) setCache(namespaces *[]namespace) {
	nsc.namespaces = namespaces
}

// getNamespaceUID returns the namespace uid given a valid namespace as argument
func (nsc *watchNamespacesCache) getNamespaceUID(namespace string) string {
	for _, ns := range *nsc.namespaces {
		if ns.name == namespace {
			return ns.uid
		}
	}
	return ""
}

// resetCache empties the namespaces field
func (nsc *watchNamespacesCache) resetCache() {
	nsc.setCache(nil)
}

// getWatchNamespaces returns the namespaces field with a list of namespaces structs
// If the field was not set, it will populate it with objects from given client
// If the ignorePattern field is set, it will filter the matches
func (nsc *watchNamespacesCache) getWatchNamespaces(ctx context.Context, c client.Client) (*[]namespace, error) {
	if nsc.namespaces == nil {
		namespaceList := corev1.NamespaceList{}
		if err := c.List(ctx, &namespaceList); err != nil {
			return nil, fmt.Errorf("listing %s: %w", namespaceList.GroupVersionKind().String(), err)
		}

		watchNamespaces := getFormattedNamespaces(namespaceList, nsc.ignorePattern)
		nsc.setCache(&watchNamespaces)
	}

	return nsc.namespaces, nil
}
