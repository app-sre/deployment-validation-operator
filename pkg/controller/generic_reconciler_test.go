package controller

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func Test_getAppLabel(t *testing.T) {
	tests := []struct {
		testName      string
		object        runtime.Object
		expectedLabel string
		expectedError error
	}{
		{
			testName: "Pod with defined label",
			object: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "app-A",
					Labels: map[string]string{
						"app": "app-A",
					},
				},
			},
			expectedLabel: "app-A",
			expectedError: nil,
		},
		{
			testName: "Pod with undefined label",
			object: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-app",
					Namespace: "test",
				},
			},
			expectedLabel: "",
			expectedError: fmt.
				Errorf("can't find any 'app' label for empty-app resource from test namespace"),
		},
		{
			testName: "PDB with defined selector label",
			object: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pdb-A",
					Namespace: "test",
				},
				Spec: policyv1.PodDisruptionBudgetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "app-with-pdb",
						},
					},
				},
			},
			expectedLabel: "app-with-pdb",
			expectedError: nil,
		},
		{
			testName: "PDB with empty selector",
			object: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-app",
					Namespace: "test",
				},
			},
			expectedLabel: "",
			expectedError: fmt.
				Errorf("can't find any 'app' label for empty-app resource from test namespace"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.object)
			assert.NoError(t, err)
			u := &unstructured.Unstructured{
				Object: o,
			}
			label, err := getAppLabel(u)
			if tt.expectedError != nil {
				assert.Error(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedLabel, label)
		})
	}
}

func Test_groupAppObjects(t *testing.T) {
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
				"B": {"test-B-deployment"},
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
				"A": {"test-A-pdb", "test-A-deployment"},
				"B": {"test-B-pod", "test-B-deployment"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create testing reconciler
			gr := &GenericReconciler{
				client:    clifake.NewClientBuilder().WithObjects(tt.objs...).Build(),
				discovery: kubefake.NewSimpleClientset().Discovery(),
			}
			groupMap, err := gr.groupAppObjects(context.Background(), tt.namespace, tt.gvks)
			assert.NoError(t, err)
			for expectedLabel, expectedNames := range tt.expectedNames {
				objects, ok := groupMap[expectedLabel]
				assert.True(t, ok, "can't find label %s", expectedLabel)
				actualNames := unstructuredToNames(objects)
				for _, exoectedName := range expectedNames {
					assert.Contains(t, actualNames, exoectedName)
				}
			}
		})
	}
}

func Test_unstructuredToTyped(t *testing.T) {
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

			gr := &GenericReconciler{
				client: clifake.NewClientBuilder().
					WithScheme(tt.scheme).
					Build(),
				discovery: kubefake.NewSimpleClientset().Discovery(),
			}

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
