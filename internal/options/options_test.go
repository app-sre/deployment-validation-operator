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
		assert.ElementsMatch(t, []string{expectedValue}, opt.WatchNamespaces)
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
