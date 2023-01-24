package controller

import (
	"fmt"
	"strings"

	"golang.stackrox.io/kube-linter/pkg/objectkinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type resourceSet struct {
	scheme       *runtime.Scheme
	apiResources map[schema.GroupKind]metav1.APIResource
}

// newResourceSet returns an empty set of resources related to the given scheme
// this object will expose methods to allow adding new resources to the set
// and to return a slice of 'APIResource' entities ("k8s.io/apimachinery/pkg/apis/meta/v1")
func newResourceSet(scheme *runtime.Scheme) *resourceSet {
	return &resourceSet{
		scheme:       scheme,
		apiResources: make(map[schema.GroupKind]metav1.APIResource),
	}
}

// Add will add a new resource to the current instance's set
// If the resource is not valid it will return an error
func (s *resourceSet) Add(key schema.GroupKind, val metav1.APIResource) error {
	if isSubResource(val) {
		return nil
	}

	if ok, err := isRegisteredKubeLinterKind(val); err != nil {
		return fmt.Errorf("checking if resource %s, is registered KubeLinter kind: %w", val.String(), err)
	} else if !ok {
		return nil
	}

	if !s.scheme.Recognizes(gvkFromMetav1APIResource(val)) {
		return nil
	}

	if existing, ok := s.apiResources[key]; ok {
		existing.Version = s.getPriorityVersion(existing.Group, existing.Version, val.Version)

		s.apiResources[key] = existing
	} else {
		s.apiResources[key] = val
	}
	return nil
}

// getPriorityVersion returns fixed group version if needed
func (s *resourceSet) getPriorityVersion(group, existingVer, currentVer string) string {
	gv := s.scheme.PrioritizedVersionsAllGroups()
	for _, v := range gv {
		if v.Group != group {
			continue
		}
		if v.Version == existingVer {
			return existingVer
		}
		if v.Version == currentVer {
			return currentVer
		}
	}
	return existingVer
}

// ToSlice returns resources from the set as a slice
// of 'APIResource' entities ("k8s.io/apimachinery/pkg/apis/meta/v1")
func (s *resourceSet) ToSlice() []metav1.APIResource {
	res := make([]metav1.APIResource, 0, len(s.apiResources))

	for _, v := range s.apiResources {
		res = append(res, v)
	}

	return res
}

func reconcileResourceList(client discovery.DiscoveryInterface,
	scheme *runtime.Scheme) ([]metav1.APIResource, error) {
	set := newResourceSet(scheme)

	_, apiResourceLists, err := client.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return nil, err
		}
		for _, rsc := range apiResourceList.APIResources {
			rsc.Group, rsc.Version = gv.Group, gv.Version
			key := schema.GroupKind{
				Group: gv.Group,
				Kind:  rsc.Kind,
			}
			if err := set.Add(key, rsc); err != nil {
				return nil, fmt.Errorf("adding resource %s to set: %w", rsc.String(), err)
			}
		}
	}
	return set.ToSlice(), nil
}

// isSubResource returns true if the apiResource.Name has a "/" in it eg: pod/status
func isSubResource(apiResource metav1.APIResource) bool {
	return strings.Contains(apiResource.Name, "/")
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
