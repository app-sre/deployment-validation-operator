package utils

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
)

// AppSelector is a helper type which
// consists of LabelSelectorOperator type and
// string values
type AppSelector struct {
	Operator metav1.LabelSelectorOperator
	Values   sets.Set[string]
}

// HasEmptySelector checks whether of the "spec.selector" and "spec.podSelector" is defined, but empty.
func HasEmptySelector(object *unstructured.Unstructured) bool {
	selector, found, _ := unstructured.NestedMap(object.Object, "spec", "selector")
	if found && len(selector) == 0 {
		return true
	}
	podSelector, found, _ := unstructured.NestedMap(object.Object, "spec", "podSelector")
	if found && len(podSelector) == 0 {
		return true
	}
	return false
}

// GetAppSelectors tries to get values (there can be more) of the "app" label.
// First it tries to read "metadata.labels.app" path (e.g for Deployments) if not found,
// then it tries to read "spec.selector.matchLabels.app" path (e.g for PodDisruptionBudget) if not found,
// then it tries to read "spec.selector.matchExpressions" path.
func GetAppSelectors(object *unstructured.Unstructured) ([]AppSelector, error) {
	appLabel, found, err := unstructured.NestedString(object.Object, "metadata", "labels", "app")
	if err != nil {
		return nil, err
	}
	if found {
		return []AppSelector{
			{
				Operator: metav1.LabelSelectorOpIn,
				Values:   sets.New(appLabel),
			},
		}, nil
	}
	// if not found try spec.selector.matchLabels path - e.g for PDB resource
	appLabel, found, err = unstructured.NestedString(object.Object,
		"spec", "selector", "matchLabels", "app")
	if err != nil {
		return nil, err
	}
	if found {
		return []AppSelector{
			{
				Operator: metav1.LabelSelectorOpIn,
				Values:   sets.New(appLabel),
			},
		}, nil
	}
	// if not found try spec.selector.matchExpressions path
	expressions, found, err := unstructured.NestedSlice(object.Object, "spec", "selector", "matchExpressions")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("can't find any 'app' label for %s resource from %s namespace",
			object.GetName(), object.GetNamespace())
	}
	appSelectors := parseMatchExpressions(expressions)
	return appSelectors, nil
}

// parseMatchExpressions tries to parse provided untyped slice of expressions
// and return a slice of appSelectors. Any expression key with a value other than "app" is skipped.
// Label selector operator "DoesNotExist" is skipped too.
func parseMatchExpressions(expressions []interface{}) []AppSelector {
	appSelectors := []AppSelector{}
	for _, exp := range expressions {
		expAsMap, ok := exp.(map[string]interface{})
		if !ok {
			continue
		}
		if expAsMap["key"] != "app" {
			continue
		}
		appSelector := AppSelector{}
		switch expAsMap["operator"] {
		case "In":
			values, _, err := unstructured.NestedStringSlice(expAsMap, "values")
			if err != nil {
				continue
			}
			appSelector.Operator = metav1.LabelSelectorOpIn
			appSelector.Values = sets.New(values...)
		case "NotIn":
			values, _, err := unstructured.NestedStringSlice(expAsMap, "values")
			if err != nil {
				continue
			}
			appSelector.Operator = metav1.LabelSelectorOpNotIn
			appSelector.Values = sets.New(values...)
		case "Exists":
			appSelector.Operator = metav1.LabelSelectorOpExists
		case "DoesNotExist":
			continue
		}
		appSelectors = append(appSelectors, appSelector)
	}
	return appSelectors
}
