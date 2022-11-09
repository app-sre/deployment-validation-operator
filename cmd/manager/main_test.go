package main

import (
	"os"
	"testing"

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
		_, err := getManagerOptions(mockScheme, options{})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, errWatchNamespaceNotSet, err)
	})

	t.Run("OK : Scheme and Namespace are set up correctly", func(t *testing.T) {
		// Given
		mockNamespace := "test"
		mockOptions := options{
			watchNamespace: &mockNamespace,
		}

		// When
		options, err := getManagerOptions(mockScheme, mockOptions)

		// Assert
		assert.NotErrorIs(t, err, errWatchNamespaceNotSet)
		assert.Equal(t, options.Namespace, mockNamespace)
		assert.Equal(t, options.Scheme, mockScheme)
	})
}

// TestOptionsStruct runs four tests on options struct methods:
//   - processEnv
//   - GetWatchNamespace
//   - GetWatchNamespace (flawed)
//   - MetricsEndpoint
func TestOptionsStruct(t *testing.T) {

	t.Run("processEnv function", func(t *testing.T) {
		// Given
		opt := options{}
		expectedValue := "test"
		os.Setenv(watchNamespaceEnvVar, expectedValue)

		// When
		opt.processEnv()

		// Assert
		assert.Equal(t, expectedValue, *opt.watchNamespace)
	})

	t.Run("GetWatchNamespace function (no result)", func(t *testing.T) {
		// Given
		opt := options{}

		// When
		_, isSet := opt.GetWatchNamespace()

		// Assert
		assert.Equal(t, false, isSet)
	})

	t.Run("GetWatchNamespace function", func(t *testing.T) {
		// Given
		expectedValue := "test"
		opt := options{
			watchNamespace: &expectedValue,
		}

		// When
		namespace, isSet := opt.GetWatchNamespace()

		// Assert
		assert.Equal(t, true, isSet)
		assert.Equal(t, expectedValue, namespace)
	})

	t.Run("MetricsEndpoint", func(t *testing.T) {
		// Given
		mockPort := int32(80)
		mockPath := "path/"
		opt := options{
			MetricsPort: mockPort,
			MetricsPath: mockPath,
		}

		// When
		endpoint := opt.MetricsEndpoint()

		// Assert
		assert.Equal(t, "http://0.0.0.0:80/path/", endpoint)
	})
}
