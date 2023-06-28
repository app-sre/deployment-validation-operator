package utils

import (
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

type manifestPath []string

var (
	metadataLabels         manifestPath = []string{"metadata", "labels", "app"}
	selectorMatchLabels    manifestPath = []string{"spec", "selector", "matchLabels", "app"}
	podSelectorMatchLabels manifestPath = []string{"spec", "podSelector", "matchLabels", "app"}

	matchLabelPaths []manifestPath = []manifestPath{metadataLabels, selectorMatchLabels, podSelectorMatchLabels}

	selectorMatchExpressions    manifestPath = []string{"spec", "selector", "matchExpressions"}
	podSelectorMatchExpressions manifestPath = []string{"spec", "podSelector", "matchExpressions"}

	matchExpressionsPaths []manifestPath = []manifestPath{selectorMatchExpressions, podSelectorMatchExpressions}
)

// GetAppSelectors tries to get values (there can be more) of the "app" label.
// It iterates over all predefined resource paths and if found then add the value
// to the slice of AppSelectors
func GetAppSelectors(object *unstructured.Unstructured) ([]AppSelector, error) {
	appSelectors := []AppSelector{}
	// iterate over all predefined resource paths
	for _, matchLabelPath := range matchLabelPaths {
		appLabel, found, err := unstructured.NestedString(object.Object, matchLabelPath...)
		if err != nil {
			continue
		}
		if found {
			appSelector := AppSelector{
				Operator: metav1.LabelSelectorOpIn,
				Values:   sets.New(appLabel),
			}
			appSelectors = append(appSelectors, appSelector)
		}
	}
	for _, matchExpressionPath := range matchExpressionsPaths {
		expressions, found, err := unstructured.NestedSlice(object.Object, matchExpressionPath...)
		if err != nil {
			continue
		}
		if !found {
			continue
		}
		appSelectors = append(appSelectors, parseMatchExpressions(expressions)...)
	}
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
