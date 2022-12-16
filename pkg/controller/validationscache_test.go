package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestValidationCache runs
func TestValidationCache(t *testing.T) {

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
}
