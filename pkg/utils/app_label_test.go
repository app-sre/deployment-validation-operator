package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			expectedLabelSelectors: []AppSelector{},
			expectedError:          nil,
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
			expectedLabelSelectors: []AppSelector{},
			expectedError:          nil,
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
