package controller

import (
	"context"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WatchNamespacesCache struct {
	cfg        watchNamespacesCacheConfig
	namespaces *[]Namespace
}

func NewWatchNamespacesCache(opts ...watchNamespacesCacheOption) *WatchNamespacesCache {
	var cfg watchNamespacesCacheConfig

	cfg.Option(opts...)

	return &WatchNamespacesCache{
		cfg: cfg,
	}
}

func (nsc *WatchNamespacesCache) setCache(namespaces *[]Namespace) {
	nsc.namespaces = namespaces
}

func (nsc *WatchNamespacesCache) GetNamespaceUID(namespace string) string {
	for _, ns := range *nsc.namespaces {
		if ns.Name == namespace {
			return ns.UID
		}
	}
	return ""
}

func (nsc *WatchNamespacesCache) Reset() {
	nsc.setCache(nil)
}

func (nsc *WatchNamespacesCache) GetWatchNamespaces(ctx context.Context, c client.Client) (*[]Namespace, error) {
	if nsc.namespaces != nil {
		return nsc.namespaces, nil
	}
	namespaceList := corev1.NamespaceList{}
	if err := c.List(ctx, &namespaceList); err != nil {
		return nil, fmt.Errorf("listing %s: %w", namespaceList.GroupVersionKind().String(), err)
	}
	watchNamespaces := []Namespace{}
	for _, ns := range namespaceList.Items {
		name := ns.GetName()
		if nsc.cfg.IgnorePattern != nil && nsc.cfg.IgnorePattern.MatchString(name) {
			continue
		}
		watchNamespaces = append(watchNamespaces, Namespace{UID: string(ns.GetUID()), Name: name})
	}
	nsc.setCache(&watchNamespaces)
	return nsc.namespaces, nil
}

type watchNamespacesCacheConfig struct {
	IgnorePattern *regexp.Regexp
}

func (c *watchNamespacesCacheConfig) Option(opts ...watchNamespacesCacheOption) {
	for _, opt := range opts {
		opt.ConfigureWatchNamespacesCache(c)
	}
}

type watchNamespacesCacheOption interface {
	ConfigureWatchNamespacesCache(c *watchNamespacesCacheConfig)
}
