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

type hashableAPIResource struct {
	// name is the plural name of the resource.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// singularName is the singular name of the resource.  This allows clients to handle
	// plural and singular opaquely.
	// The singularName is more correct for reporting status on a single item and both
	// singular and plural are allowed from the kubectl CLI interface.
	SingularName string `json:"singularName" protobuf:"bytes,6,opt,name=singularName"`
	// namespaced indicates if a resource is namespaced or not.
	Namespaced bool `json:"namespaced" protobuf:"varint,2,opt,name=namespaced"`
	// group is the preferred group of the resource.  Empty implies the group of the
	// containing resource list.
	// For subresources, this may have a different value, for example: Scale".
	Group string `json:"group,omitempty" protobuf:"bytes,8,opt,name=group"`
	// version is the preferred version of the resource.  Empty implies the version of the
	// containing resource list
	// For subresources, this may have a different value,
	//for example: v1 (while inside a v1beta1 version of the core resource's group)".
	Version string `json:"version,omitempty" protobuf:"bytes,9,opt,name=version"`
	// kind is the kind for the resource (e.g. 'Foo' is the kind for a resource 'foo')
	Kind string `json:"kind" protobuf:"bytes,3,opt,name=kind"`
}

func (hr *hashableAPIResource) gvk() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   hr.Group,
		Version: hr.Version,
		Kind:    hr.Kind,
	}
}

func newResourceSet(scheme *runtime.Scheme) *resourceSet {
	return &resourceSet{
		scheme:       scheme,
		apiResources: make(map[schema.GroupKind]metav1.APIResource),
	}
}

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

func (s *resourceSet) ToSlice() []hashableAPIResource {
	res := make([]hashableAPIResource, 0, len(s.apiResources))

	for _, v := range s.apiResources {
		hashable := hashableAPIResource{
			Name:         v.Name,
			SingularName: v.SingularName,
			Namespaced:   v.Namespaced,
			Group:        v.Group,
			Version:      v.Version,
			Kind:         v.Kind,
		}
		res = append(res, hashable)
	}

	return res
}

func reconcileResourceList(client discovery.DiscoveryInterface,
	scheme *runtime.Scheme) ([]hashableAPIResource, error) {
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
