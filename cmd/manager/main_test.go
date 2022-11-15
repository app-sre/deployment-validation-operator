package main

import (
	"testing"

	"github.com/app-sre/deployment-validation-operator/internal/options"
	"github.com/stretchr/testify/assert"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

// TestGetManagerOptionsFn run two tests:
// - A controlled error (declared on the package)
// - A normal run, checking namespace and scheme are set up correctly
func TestGetManagerOptionsFn(t *testing.T) {
	// Given
	mockScheme := k8sruntime.NewScheme()

	t.Run("Error : WatchNamespace not set", func(t *testing.T) {
		// When
		_, err := getManagerOptions(mockScheme, options.Options{})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, errWatchNamespaceNotSet, err)
	})

	t.Run("OK : Scheme and Namespace are set up correctly", func(t *testing.T) {
		// Given
		mockNamespace := "test"
		mockOptions := options.Options{
			WatchNamespace: &mockNamespace,
		}

		// When
		options, err := getManagerOptions(mockScheme, mockOptions)

		// Assert
		assert.NotErrorIs(t, err, errWatchNamespaceNotSet)
		assert.Equal(t, options.Namespace, mockNamespace)
		assert.Equal(t, options.Scheme, mockScheme)
	})
}
