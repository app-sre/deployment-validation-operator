package options

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOptionsStruct runs four tests on options struct methods:
//   - processEnv
//   - GetWatchNamespace
//   - GetWatchNamespace (flawed)
//   - MetricsEndpoint
func TestOptionsStruct(t *testing.T) {

	t.Run("processEnv function", func(t *testing.T) {
		// Given
		opt := Options{}
		expectedValue := "test"
		os.Setenv(watchNamespaceEnvVar, expectedValue)

		// When
		opt.processEnv()

		// Assert
		assert.Equal(t, expectedValue, *opt.WatchNamespace)
	})

	t.Run("GetWatchNamespace function (no result)", func(t *testing.T) {
		// Given
		opt := Options{}

		// When
		_, isSet := opt.GetWatchNamespace()

		// Assert
		assert.Equal(t, false, isSet)
	})

	t.Run("GetWatchNamespace function", func(t *testing.T) {
		// Given
		expectedValue := "test"
		opt := Options{
			WatchNamespace: &expectedValue,
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
		opt := Options{
			MetricsPort: mockPort,
			MetricsPath: mockPath,
		}

		// When
		endpoint := opt.MetricsEndpoint()

		// Assert
		assert.Equal(t, "http://0.0.0.0:80/path/", endpoint)
	})
}
