package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetAppSelectors(t *testing.T) {
	tests := []struct {
		testName               string
		object                 runtime.Object
		expectedLabelSelectors []AppSelector
		expectedError          error
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
			expectedLabelSelectors: []AppSelector{
				{
					Operator: metav1.LabelSelectorOpIn,
					Values:   sets.New("app-A"),
				},
			},
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
			expectedLabelSelectors: nil,
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
			expectedLabelSelectors: []AppSelector{
				{
					Operator: metav1.LabelSelectorOpIn,
					Values:   sets.New("app-with-pdb"),
				},
			},
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
			expectedLabelSelectors: nil,
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
			appSelectors, err := GetAppSelectors(u)
			if tt.expectedError != nil {
				assert.Error(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedLabelSelectors, appSelectors)
		})
	}
}

func TestHasEmptySelector(t *testing.T) {
	tests := []struct {
		testName      string
		object        runtime.Object
		emptySelector bool
	}{
		{
			testName: "Deployment with nil selector",
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "tests",
				},
				Spec: appsv1.DeploymentSpec{},
			},
			emptySelector: false,
		},
		{
			testName: "Deployment with non-empty selector",
			object: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "tests",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test-app",
						},
					},
				},
			},
			emptySelector: false,
		},
		{
			testName: "PDB with empty selector",
			object: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "tests",
				},
				Spec: policyv1.PodDisruptionBudgetSpec{
					Selector: &metav1.LabelSelector{},
				},
			},
			emptySelector: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.object)
			assert.NoError(t, err)
			u := &unstructured.Unstructured{
				Object: o,
			}

			empty := HasEmptySelector(u)
			assert.Equal(t, tt.emptySelector, empty)
		})
	}
}
