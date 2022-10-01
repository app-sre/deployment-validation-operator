package controller

import (
	"golang.stackrox.io/kube-linter/pkg/objectkinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func reconcileResourceList(c *discovery.DiscoveryClient) ([]metav1.APIResource, error) {
	apiResourceLists, err := c.ServerPreferredResources()
	if err != nil {
		return nil, err
	}
	apiResources := []metav1.APIResource{}
	for _, apiResourceGroup := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceGroup.GroupVersion)
		if err != nil {
			return nil, err
		}
		for _, apiResource := range apiResourceGroup.APIResources {

			apiResource.Version = gv.Version
			apiResource.Group = gv.Group

			canValidate, err := isRegisteredKubeLinterKind(apiResource)
			if err != nil {
				return nil, err
			}

			if !canValidate {
				continue
			}
			apiResources = append(apiResources, apiResource)
		}
	}
	return apiResources, nil
}

func isRegisteredKubeLinterKind(rsrc metav1.APIResource) (bool, error) {
	// Construct the gvks for objects to watch.  Remove the Any
	// kind or else all objects kinds will be watched.
	kubeLinterKinds := getKubeLinterKinds()
	kubeLinterMatcher, err := objectkinds.ConstructMatcher(kubeLinterKinds...)
	if err != nil {
		return false, err
	}

	gvk := gvkFromMetav1APIResource(rsrc)
	if kubeLinterMatcher.Matches(gvk) {
		return true, nil
	}
	return false, nil
}

func getKubeLinterKinds() []string {
	kubeLinterKinds := objectkinds.AllObjectKinds()
	for i := range kubeLinterKinds {
		if kubeLinterKinds[i] == objectkinds.Any {
			kubeLinterKinds = append(kubeLinterKinds[:i], kubeLinterKinds[i+1:]...)
			break
		}
	}
	return kubeLinterKinds
}

func gvkFromMetav1APIResource(rsc metav1.APIResource) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   rsc.Group,
		Version: rsc.Version,
		Kind:    rsc.Kind,
	}
}
