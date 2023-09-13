package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGetLabelSelector(t *testing.T) {
	tests := []struct {
		name                  string
		unstructuredResource  *unstructured.Unstructured
		expectedLabelSelector *metav1.LabelSelector
	}{
		{
			name: "When no selector is defined then LabelSelector is nil",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
				},
			},
			expectedLabelSelector: nil,
		},
		{
			name: "When there's an empty selector then empty LabelSelector is obtained",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
					"spec": map[string]interface{}{
						"selector": map[string]interface{}{},
					},
				},
			},
			expectedLabelSelector: &metav1.LabelSelector{},
		},
		{
			name: "When there's an empty podSelector then empty LabelSelector is obtained",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "NetworkPolicy",
					"apiVersion": "networking.k8s.io/v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
					"spec": map[string]interface{}{
						"podSelector": map[string]interface{}{},
					},
				},
			},
			expectedLabelSelector: &metav1.LabelSelector{},
		},
		{
			name: "Non empty podSelector with matchExpressions requirements",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "NetworkPolicy",
					"apiVersion": "networking.k8s.io/v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
					"spec": map[string]interface{}{
						"podSelector": map[string]interface{}{
							"matchExpressions": []interface{}{
								map[string]interface{}{
									"key":      "app",
									"operator": "In",
									"values": []interface{}{
										"A",
										"B",
									},
								},
								map[string]interface{}{
									"key":      "environment",
									"operator": "NotIn",
									"values": []interface{}{
										"testing",
										"staging",
									},
								},
							},
						},
					},
				},
			},
			expectedLabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"A", "B"},
					},
					{
						Key:      "environment",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"testing", "staging"},
					},
				},
			},
		},
		{
			name: "Non empty selector with matchLabels requirements",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
					"spec": map[string]interface{}{
						"selector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"app":         "A",
								"environment": "production",
							},
						},
					},
				},
			},
			expectedLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "A",
					"environment": "production",
				},
			},
		},
		{
			name: "Selector with multiple requirements",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]string{
						"name":      "test-pod",
						"namespace": "test",
					},
					"spec": map[string]interface{}{
						"selector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"app":         "A",
								"environment": "production",
							},
							"matchExpressions": []interface{}{
								map[string]interface{}{
									"key":      "app",
									"operator": "Exists",
								},
								map[string]interface{}{
									"key":      "environment",
									"operator": "DoesNotExist",
								},
								map[string]interface{}{
									"key":      "test-key",
									"operator": "In",
									"values": []interface{}{
										"FOO",
										"BAR",
									},
								},
							},
						},
					},
				},
			},
			expectedLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "A",
					"environment": "production",
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "environment",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
					{
						Key:      "test-key",
						Operator: metav1.LabelSelectorOpIn,
						Values: []string{
							"FOO",
							"BAR",
						},
					},
				},
			},
		},
		{
			name: "selector defined as map[string]string with no metav1.LabelSelector",
			unstructuredResource: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Service",
					"apiVersion": "v1",
					"metadata": map[string]string{
						"name":      "test-service",
						"namespace": "test",
					},
					"spec": map[string]interface{}{
						"selector": map[string]interface{}{
							"app":         "A",
							"environment": "production",
						},
					},
				},
			},
			expectedLabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "A",
					"environment": "production",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := GetLabelSelector(tt.unstructuredResource)
			assert.Equal(t, tt.expectedLabelSelector, ls)
		})

	}

}
