package controller

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/configmap"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clifake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestHelperFunctions runs five tests on helper functions on generic_reconciler.go
// - checks that infFromEnv works as expected
// - checks that infFromEnv returns an error if the env variable is not a number
// - checks that infFromEnv returns false on the control value if the env variable is not set
// - checks that defaultOrEnv works as expected
// - checks that defaultOrEnv returns default value if env variable is not set
func TestHelperFunctions(t *testing.T) {

	t.Run("intFromEnv returns correct int", func(t *testing.T) {
		// Given
		os.Setenv("test", "1")

		// When
		test, _, _ := intFromEnv("test")

		// Assert
		assert.Equal(t, 1, test)
	})

	t.Run("intFromEnv returns error if value is not int", func(t *testing.T) {
		// Given
		os.Setenv("test", "mock")

		// When
		_, _, test := intFromEnv("test")

		// Assert
		assert.Error(t, test)
	})

	t.Run("intFromEnv returns false as control value if the env variable was not set", func(t *testing.T) {
		// When
		os.Unsetenv("test")

		// When
		_, test, _ := intFromEnv("test")

		// Assert
		assert.False(t, test)
	})

	t.Run("defaultOrEnv returns correct int", func(t *testing.T) {
		// Given
		os.Setenv("test", "1")

		// When
		test, _ := defaultOrEnv("test", 2)

		// Assert
		assert.Equal(t, 1, test)
	})

	t.Run("defaultOrEnv returns default value if no env variable was set", func(t *testing.T) {
		// When
		os.Unsetenv("test")

		// When
		test, _ := defaultOrEnv("test", 2)

		// Assert
		assert.Equal(t, 2, test)
	})
}

func TestGroupAppObjects(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		objs          []client.Object
		gvks          []schema.GroupVersionKind
		expectedNames map[string][]string
	}{
		{
			name:      "Objects from different namespace are not found",
			namespace: "test",
			objs: []client.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-B-deployment",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-B-pod",
						Namespace: "another-namespace",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-A-pod",
						Namespace: "another-namespace",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=B": {"test-B-deployment"},
			},
		},
		{
			name:      "Two groups of objects with labels app=A and app=B",
			namespace: "test",
			objs: []client.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-A-deployment",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-A-pdb",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "A",
							},
						},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-B-deployment",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-B-pod",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "policy",
					Kind:    "PodDisruptionBudget",
					Version: "v1",
				},
				{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				}},
			expectedNames: map[string][]string{
				"app=A": {"test-A-pdb", "test-A-deployment"},
				"app=B": {"test-B-pod", "test-B-deployment"},
			},
		},
		{
			name:      "Four deployments with multiple various matching PDBs",
			namespace: "test",
			objs: []client.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment-A",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment-B",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment-C",
						Namespace: "test",
						Labels: map[string]string{
							"app": "C",
						},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment-D",
						Namespace: "test",
						Labels: map[string]string{
							"app": "D",
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pdb-A-B-C",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"A", "B", "C"},
								},
							},
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pdb-not-in-C",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{"C"},
								},
							},
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pdb-exists",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpExists,
								},
							},
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pdb-not-in-D",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{"D"},
								},
							},
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "policy",
					Kind:    "PodDisruptionBudget",
					Version: "v1",
				},
				{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=A": {"test-pdb-A-B-C", "test-pdb-not-in-C", "test-pdb-exists", "test-deployment-A", "test-pdb-not-in-D"}, //nolint:lll
				"app=B": {"test-pdb-A-B-C", "test-pdb-not-in-C", "test-pdb-exists", "test-deployment-B", "test-pdb-not-in-D"}, //nolint:lll
				"app=C": {"test-pdb-A-B-C", "test-pdb-exists", "test-deployment-C", "test-pdb-not-in-D"},                      //nolint:lll
				"app=D": {"test-deployment-D", "test-pdb-exists", "test-pdb-not-in-C"},
			},
		},
		{
			name:      "Two StatefulSets with some matching PDBs",
			namespace: "test",
			objs: []client.Object{
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pdb-not-in-A",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{"A"},
								},
							},
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pdb-not-in-B",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{"B"},
								},
							},
						},
					},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "statefulset-A",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "statefulset-B",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "apps",
					Kind:    "StatefulSet",
					Version: "v1",
				},
				{
					Group:   "policy",
					Kind:    "PodDisruptionBudget",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=A": {"statefulset-A", "pdb-not-in-B"},
				"app=B": {"statefulset-B", "pdb-not-in-A"},
			},
		},
		{
			name:      "Two Deployments and a PDB with empty selector matching both deployments",
			namespace: "test",
			objs: []client.Object{
				&policyv1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pdb",
						Namespace: "test",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-A",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-B",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				{
					Group:   "policy",
					Kind:    "PodDisruptionBudget",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=A": {"app-A", "test-pdb"},
				"app=B": {"app-B", "test-pdb"},
			},
		},
		{
			name:      "Deployment with matching NetworkPolicy",
			namespace: "test",
			objs: []client.Object{
				&networkingv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "np-A",
						Namespace: "test",
					},
					Spec: networkingv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpNotIn,
									Values:   []string{"B"},
								},
							},
						},
					},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deployment-A",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				{
					Group:   "networking.k8s.io",
					Kind:    "NetworkPolicy",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=A": {"deployment-A", "np-A"},
			},
		},
		{
			name:      "Deployment with non-matching NetworkPolicy (no selector)",
			namespace: "test",
			objs: []client.Object{
				&networkingv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "np-A",
						Namespace: "test",
					},
					Spec: networkingv1.NetworkPolicySpec{},
				},
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deployment-A",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "apps",
					Kind:    "Deployment",
					Version: "v1",
				},
				{
					Group:   "networking.k8s.io",
					Kind:    "NetworkPolicy",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=A": {"deployment-A"},
			},
		},
		{
			name:      "Pods with different matching Services",
			namespace: "test",
			objs: []client.Object{
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service-A",
						Namespace: "test",
					},
					Spec: v1.ServiceSpec{
						Selector: map[string]string{
							"app": "A",
						},
					},
				},
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "service-B",
						Namespace: "test",
					},
					Spec: v1.ServiceSpec{
						Selector: map[string]string{
							"app": "B",
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-A",
						Namespace: "test",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
				&v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-B",
						Namespace: "test",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "",
					Kind:    "Service",
					Version: "v1",
				},
				{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
			expectedNames: map[string][]string{
				"app=A": {"pod-A", "service-A"},
				"app=B": {"pod-B", "service-B"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create testing reconciler
			gr, err := createTestReconciler(nil, tt.objs)
			assert.NoError(t, err)
			groupMap, err := gr.groupAppObjects(context.Background(), tt.namespace, tt.gvks)
			assert.NoError(t, err)

			for expectedLabel, expectedNames := range tt.expectedNames {
				objects, ok := groupMap[expectedLabel]
				assert.True(t, ok, "can't find label %s", expectedLabel)
				actualNames := unstructuredToNames(objects)
				for _, expectedName := range expectedNames {
					assert.Contains(t, actualNames, expectedName,
						"can't find %s for label value %s", expectedName, expectedLabel)
				}
			}
		})
	}
}

func TestUnstructuredToTyped(t *testing.T) {
	tests := []struct {
		name          string
		scheme        *runtime.Scheme
		u             *unstructured.Unstructured
		expectedName  string
		expectedGVK   schema.GroupVersionKind
		expectedError error
	}{
		{
			name:   "Simple Pod as unstructured is properly typed",
			scheme: runtime.NewScheme(),
			u: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
				},
			},
			expectedName: "test-pod",
			expectedGVK: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			expectedError: nil,
		},
		{
			name:   "invalid unstructured is not type",
			scheme: runtime.NewScheme(),
			u: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Unknown",
					"apiVersion": "Unknown",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
				},
			},
			expectedName: "",
			expectedGVK: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			expectedError: fmt.
				Errorf("looking up object type: creating new object of type /Unknown, Kind=Unknown"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v1.AddToScheme(tt.scheme)
			assert.NoError(t, err)

			gr, err := createTestReconciler(tt.scheme, nil)
			assert.NoError(t, err)
			o, err := gr.unstructuredToTyped(tt.u)
			if tt.expectedError == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedName, o.GetName())
				assert.Equal(t, tt.expectedGVK, o.GetObjectKind().GroupVersionKind())
			} else {
				assert.ErrorContains(t, err, tt.expectedError.Error())
			}
		})
	}
}

// unstructuredToNames iterates over slice of unstructured objects and
// returns a slice of their names
func unstructuredToNames(objs []*unstructured.Unstructured) []string {
	names := []string{}
	for _, o := range objs {
		names = append(names, o.GetName())
	}
	return names
}

func TestGetNamespacedResourcesGVK(t *testing.T) {
	unitTests := []struct {
		name   string
		arg    []metav1.APIResource
		result []schema.GroupVersionKind
	}{
		{
			name: "Received resource is on a namespace and it's returned as GVK",
			arg: []metav1.APIResource{{
				Group: "test", Version: "v1", Kind: "Ingress", Namespaced: true,
			}},
			result: []schema.GroupVersionKind{{
				Group: "test", Version: "v1", Kind: "Ingress",
			}},
		},
		{
			name: "Received resource is not on a namespace and it's not returned as GVK",
			arg: []metav1.APIResource{{
				Group: "test", Version: "v1", Kind: "Ingress", Namespaced: false,
			}},
			result: []schema.GroupVersionKind{},
		},
	}

	for _, ut := range unitTests {
		t.Run(ut.name, func(t *testing.T) {
			// Given
			gr := GenericReconciler{}

			// When
			test := gr.getNamespacedResourcesGVK(ut.arg)

			// Assert
			assert.Equal(t, ut.result, test)
		})
	}
}

func TestProcessNamespacedResources(t *testing.T) {
	tests := []struct {
		name       string
		objects    []client.Object
		gvks       []schema.GroupVersionKind
		namespaces *[]namespace
	}{
		{
			name: "basic test with objects from two namespaces",
			objects: []client.Object{

				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-A-deployment",
						Namespace: "test-A",
						UID:       "depA",
						Labels: map[string]string{
							"app": "A",
						},
					},
				},
				&policyv1.PodDisruptionBudget{
					TypeMeta: metav1.TypeMeta{
						Kind:       "PodDisruptionBudget",
						APIVersion: "policy/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-A-pdb",
						UID:       "pdbA",
						Namespace: "test-A",
					},
					Spec: policyv1.PodDisruptionBudgetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "A",
							},
						},
					},
				},
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-B-deployment",
						UID:       "depB",
						Namespace: "test-B",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
				&v1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-B-pod",
						UID:       "podB",
						Namespace: "test-B",
						Labels: map[string]string{
							"app": "B",
						},
					},
				},
			},
			gvks: []schema.GroupVersionKind{
				{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
				{
					Group:   "policy",
					Kind:    "PodDisruptionBudget",
					Version: "v1",
				},
			},
			namespaces: &[]namespace{
				{
					uid:  "A",
					name: "test-A",
				},
				{
					uid:  "B",
					name: "test-B",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// register all the required schemes
			sch := runtime.NewScheme()
			err := appsv1.AddToScheme(sch)
			assert.NoError(t, err)
			err = v1.AddToScheme(sch)
			assert.NoError(t, err)
			err = policyv1.AddToScheme(sch)
			assert.NoError(t, err)

			testReconciler, err := createTestReconciler(sch, tt.objects)
			assert.NoError(t, err)

			// set some namespaces to be watched
			testReconciler.watchNamespaces.setCache(tt.namespaces)
			err = testReconciler.processNamespacedResources(context.Background(), tt.gvks, tt.namespaces)
			assert.NoError(t, err)
			for _, o := range tt.objects {
				vr, ok := testReconciler.objectValidationCache.retrieve(o)
				assert.True(t, ok, "can't find object %v in the validation cache", o)
				assert.Equal(t, string(o.GetUID()), vr.uid)

				co, ok := testReconciler.currentObjects.retrieve(o)
				assert.True(t, ok, "can't find object %v in the current objects", o)
				assert.Equal(t, string(o.GetUID()), co.uid)
			}
		})
	}
}

func TestHandleResourceDeletions(t *testing.T) {
	tests := []struct {
		name                     string
		testNamespaces           []namespace
		testCurrentObjects       []client.Object
		testValidatedObjects     []client.Object
		expectedValidatedObjects []client.Object
	}{
		{
			name: "Same objects in 'currentObjects' cache and 'objectValidationCache' cache",
			testNamespaces: []namespace{
				{
					uid:  "A",
					name: "test-A",
				},
			},
			testCurrentObjects: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep-A",
						Namespace: "test-A",
						UID:       "uidA",
					},
				},
			},
			testValidatedObjects: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep-A",
						Namespace: "test-A",
						UID:       "uidA",
					},
				},
			},
			expectedValidatedObjects: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep-A",
						Namespace: "test-A",
						UID:       "uidA",
					},
				},
			},
		},
		{
			name: "All objects that are not in 'currentObjects' cache are removed from the 'objectValidationCache'", //nolint:lll
			testNamespaces: []namespace{
				{
					uid:  "A",
					name: "test-A",
				},
				{
					uid:  "B",
					name: "test-B",
				},
			},
			testCurrentObjects: []client.Object{},
			testValidatedObjects: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep-A",
						Namespace: "test-A",
						UID:       "uidA",
					},
				},
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dep-B",
						Namespace: "test-B",
						UID:       "uidB",
					},
				},
			},
			expectedValidatedObjects: nil,
		},
	}

	for _, testCase := range tests {
		tt := testCase
		t.Run(tt.name, func(t *testing.T) {
			testReconciler, err := createTestReconciler(nil, nil)
			assert.NoError(t, err)
			testReconciler.watchNamespaces.setCache(&tt.testNamespaces)

			// store the test objects in the caches
			for _, co := range tt.testCurrentObjects {
				testReconciler.currentObjects.store(co, validations.ObjectNeedsImprovement)
			}
			for _, co := range tt.testValidatedObjects {
				testReconciler.objectValidationCache.store(co, validations.ObjectNeedsImprovement)
			}
			testReconciler.handleResourceDeletions()
			// currentObjects should be always empty after calling handleResourceDeletions
			for _, co := range tt.testCurrentObjects {
				_, ok := testReconciler.currentObjects.retrieve(co)
				assert.False(t, ok)
			}

			if tt.expectedValidatedObjects == nil {
				for _, vo := range tt.testValidatedObjects {
					_, ok := testReconciler.objectValidationCache.retrieve(vo)
					assert.False(t, ok)
				}
			} else {
				for _, vo := range tt.expectedValidatedObjects {
					_, ok := testReconciler.objectValidationCache.retrieve(vo)
					assert.True(t, ok)
				}
			}
		})
	}
}

func TestListLimit(t *testing.T) {
	os.Setenv(EnvResorucesPerListQuery, "2")
	testReconciler, err := createTestReconciler(runtime.NewScheme(), nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), testReconciler.listLimit)
}

func createTestReconciler(scheme *runtime.Scheme, objects []client.Object) (*GenericReconciler, error) {
	cliBuilder := clifake.NewClientBuilder()
	if scheme != nil {
		cliBuilder.WithScheme(scheme)
	}
	if objects != nil {
		cliBuilder.WithObjects(objects...)
	}
	client := cliBuilder.Build()
	cli := kubefake.NewSimpleClientset()

	ve, err := validations.NewValidationEngine("", make(map[string]*prometheus.GaugeVec))
	if err != nil {
		return nil, err
	}
	return NewGenericReconciler(client, cli.Discovery(), &configmap.Watcher{}, ve)
}

// BenchmarkGroupAppObjects measures the performance of grouping Kubernetes objects based on their labels.
// The benchmark focuses on a scenario where a Reconciler needs to group different objects based on the 'app' label.
// # Benchmark configuration:
//
// The benchmark is configured with the following parameters:
// - namespace: "test" - The namespace in which the objects will be created.
// - deploymentNumber: 100 - The number of Deployment objects to create to test.
// - channelBuffer: 10 - The buffer size for the channel used in the grouping process.
//
// # How to run:
//
// To run the benchmark, run the following command:
//
//	go test ./pkg/controller/ -bench ^BenchmarkGroupAppObjects$
//
// Adjust the number of deployments or the buffer to match a given scenario.
// Note that the local benchmark can be modified by other processes running in the background.
// If the 'groupAppObjects' function is modified, run some benchmarks with before and after status
// to compare performance improvements.
func BenchmarkGroupAppObjects(b *testing.B) {
	namespace := "test"
	deploymentNumber := 100

	// Given
	objs := []client.Object{
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pdb-A-B-C", Namespace: namespace},
			Spec: policyv1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key: "app", Operator: metav1.LabelSelectorOpIn,
							Values: []string{"A", "B", "C"},
						},
					},
				},
			},
		},
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pdb-not-in-C", Namespace: namespace},
			Spec: policyv1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key: "app", Operator: metav1.LabelSelectorOpNotIn,
							Values: []string{"C"},
						},
					},
				},
			},
		},
	}

	objs = append(objs, generateDeployments(deploymentNumber, namespace)...)

	gvks := []schema.GroupVersionKind{
		{
			Group: "policy", Kind: "PodDisruptionBudget", Version: "v1",
		},
		{
			Group: "apps", Kind: "Deployment", Version: "v1",
		},
	}

	// When
	gr, err := createTestReconciler(nil, gvks, objs)
	assert.NoError(b, err)

	// Benchmark
	for i := 0; i < b.N; i++ {
		ch := make(chan groupOfObjects)
		go gr.groupAppObjects(context.Background(), namespace, ch)

		for group := range ch {
			b.Logf("Received group: %s, with %d objects", group.label, len(group.objects))
		}
	}

	// This line reports memory consumption automatically (Bytes/operation and number of allocations/op)
	b.ReportAllocs()
}

// generateDeployments is a helper function for benchmark to create iterative deployments
func generateDeployments(count int, namespace string) []client.Object {
	objects := make([]client.Object, count)

	for i := 0; i < count; i++ {
		objects[i] = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-deployment-%d", i),
				Namespace: namespace,
				Labels: map[string]string{
					"app": []string{"A", "B", "C", "D"}[rand.Intn(4)],
				},
			},
		}
	}

	return objects
}
