package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type manifestPath []string

var (
	// known manifest/resource paths for selector requirements
	selectorMatchLabels         manifestPath = []string{"spec", "selector", "matchLabels"}
	podSelectorMatchLabels      manifestPath = []string{"spec", "podSelector", "matchLabels"}
	selectorMatchExpressions    manifestPath = []string{"spec", "selector", "matchExpressions"}
	podSelectorMatchExpressions manifestPath = []string{"spec", "podSelector", "matchExpressions"}
)

// GetLabelSelector parses the unstructured object and checks following:
// 1. Is there some "spec.selector" or "spec.podSelector". If not then return nil.
// 2. Is there some empty "spec.selector" or "spec.podSelector". If yes then return empty LabelSelector struct.
// 3. Otherwise try to parse "matchLabels" and "matchExpressions" requirements and create LabelSelector with
// these values.
func GetLabelSelector(object *unstructured.Unstructured) *metav1.LabelSelector {
	if !hasSelector(object) {
		return nil
	}

	if hasEmptySelector(object) {
		return &metav1.LabelSelector{}
	}

	matchLabels := readMatchLabels(object)
	matchExpressions := readMatchExpressions(object)
	return &metav1.LabelSelector{
		MatchExpressions: matchExpressions,
		MatchLabels:      matchLabels,
	}
}

// readMatchLabels tries to parse known paths for the "matchLabels" attribute in the
// unstructured object input
func readMatchLabels(object *unstructured.Unstructured) map[string]string {
	for _, path := range []manifestPath{selectorMatchLabels, podSelectorMatchLabels} {
		pathExists := pathExistsAsMap(object, path)
		if pathExists {
			matchLabels, _, _ := unstructured.NestedStringMap(object.Object, path...)
			return matchLabels
		}
	}
	return nil
}

// readMatchExpressions tries to parse known paths for the "matchExpressions" attribute in the
// unstructured object input
func readMatchExpressions(object *unstructured.Unstructured) []metav1.LabelSelectorRequirement {
	matchExpressionsRequirements := []metav1.LabelSelectorRequirement{}
	for _, path := range []manifestPath{selectorMatchExpressions, podSelectorMatchExpressions} {
		pathExists := pathExistsAsSlice(object, path)
		if !pathExists {
			continue
		}
		matchExpressions, _, _ := unstructured.NestedSlice(object.Object, path...)
		for _, matchExpression := range matchExpressions {
			meMap := matchExpression.(map[string]interface{})
			labelSelectorReq := metav1.LabelSelectorRequirement{}
			for k, v := range meMap {
				switch {
				case k == "key":
					labelSelectorReq.Key = v.(string)
				case k == "operator":
					labelSelectorReq.Operator = metav1.LabelSelectorOperator(v.(string))
				case k == "values":
					stringValues := []string{}
					for _, value := range v.([]interface{}) {
						stringValues = append(stringValues, value.(string))
					}
					labelSelectorReq.Values = stringValues
				}
			}
			matchExpressionsRequirements = append(matchExpressionsRequirements, labelSelectorReq)
		}
	}

	if len(matchExpressionsRequirements) == 0 {
		return nil
	}
	return matchExpressionsRequirements
}

// hasEmptySelector checks whether of the "spec.selector" and "spec.podSelector" is defined, but empty.
func hasEmptySelector(object *unstructured.Unstructured) bool {
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

// hasSelector checks whether there is some "spec.selector" or "spec.podSelector" value defined.
func hasSelector(object *unstructured.Unstructured) bool {
	_, selectorFound, _ := unstructured.NestedMap(object.Object, "spec", "selector")
	_, podSelectorfound, _ := unstructured.NestedMap(object.Object, "spec", "podSelector")
	return selectorFound || podSelectorfound
}

// pathExistsAsMap checks whether provided manifest path exists as map[string]interface{}
func pathExistsAsMap(object *unstructured.Unstructured, path manifestPath) bool {
	_, found, _ := unstructured.NestedMap(object.Object, path...)
	return found
}

// pathExistsAsSlice checks whether provided manifest path exists as []interface{}
func pathExistsAsSlice(object *unstructured.Unstructured, path manifestPath) bool {
	_, found, _ := unstructured.NestedSlice(object.Object, path...)
	return found
}
