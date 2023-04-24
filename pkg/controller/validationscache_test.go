package controller

import (
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestValidationCache runs four tests on validationscache file's functions
// checks that store adds key-value pair properly
// checks objectAlreadyValidated different scenarios
// - key does not exist
// - resource version does not match
// - everything runs properly
func TestValidationsCache(t *testing.T) {

	t.Run("store adds new key and value to current instance", func(t *testing.T) {
		// Given
		mock := newValidationCache()
		mockClientObject := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "mock_version",
			UID:             "mock_uid",
		}}

		// When
		mock.store(&mockClientObject, "mock_outcome")

		// Assert
		expected := newValidationResource(newResourceversionVal("mock_version"), "mock_uid", "mock_outcome")
		assert.Equal(t, expected, (*mock)[newValidationKey(&mockClientObject)])
	})

	t.Run("objectAlreadyValidated : key does not exist", func(t *testing.T) {
		// Given
		mock := newValidationCache()
		mockClientObject := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "mock_version",
			UID:             "mock_uid",
		}}

		// When
		test := mock.objectAlreadyValidated(&mockClientObject)

		// Assert
		assert.False(t, test)
	})

	t.Run("objectAlreadyValidated : resource versions do not match", func(t *testing.T) {
		// Given
		mock := newValidationCache()
		mockClientObject := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "mock_version",
			UID:             "mock_uid",
		}}
		mock.store(&mockClientObject, "mock_outcome")
		toBeRemovedKey := newValidationKey(&mockClientObject)

		// When
		mockClientObject.ResourceVersion = "mock_new_version"
		test := mock.objectAlreadyValidated(&mockClientObject)

		// Assert
		assert.False(t, test)
		assert.False(t, mock.has(toBeRemovedKey))
	})

	t.Run("objectAlreadyValidated : OK", func(t *testing.T) {
		// Given
		mock := newValidationCache()
		mockClientObject := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "mock_version",
			UID:             "mock_uid",
		}}
		mock.store(&mockClientObject, "mock_outcome")

		// When
		test := mock.objectAlreadyValidated(&mockClientObject)

		// Assert
		assert.True(t, test)
	})

	t.Run("storing two different objects with the same name and namespace", func(t *testing.T) {
		// Given
		testCache := newValidationCache()
		dep1 := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-app",
				UID:       "foo123",
			},
		}
		dep2 := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "test-app",
				UID:       "bar345",
			},
		}
		testCache.store(&dep1, validations.ObjectNeedsImprovement)
		testCache.store(&dep2, validations.ObjectValid)

		resource1, exists := testCache.retrieve(&dep1)
		assert.True(t, exists)
		assert.Equal(t, validations.ObjectNeedsImprovement, resource1.outcome)

		resource2, exists := testCache.retrieve(&dep2)
		assert.True(t, exists)
		assert.Equal(t, validations.ObjectValid, resource2.outcome)
	})
}
