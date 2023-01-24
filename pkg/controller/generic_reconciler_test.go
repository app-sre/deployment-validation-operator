package controller

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHelperFunctions runs five tests on helper functions on generic_reconciler.go
// - checks that infFromEnv works as expected
// - checks that infFromEnv returns an error if the env variable is not a number
// - checks that infFromEnv returns false on the control value if the env variable is not set
// - checks that defaultOrEnv works as expected
// - checks that defaultOrEnv returns default value if env variable is not set
func TestHelperFunctions(t *testing.T) {

	t.Run("intFromEnv returns correct int", func(t *testing.T) {
		// Given
		os.Setenv("test", "1")

		// When
		test, _, _ := intFromEnv("test")

		// Assert
		assert.Equal(t, 1, test)
	})

	t.Run("intFromEnv returns error if value is not int", func(t *testing.T) {
		// Given
		os.Setenv("test", "mock")

		// When
		_, _, test := intFromEnv("test")

		// Assert
		assert.Error(t, test)
	})

	t.Run("intFromEnv returns false as control value if the env variable was not set", func(t *testing.T) {
		// When
		os.Unsetenv("test")

		// When
		_, test, _ := intFromEnv("test")

		// Assert
		assert.False(t, test)
	})

	t.Run("defaultOrEnv returns correct int", func(t *testing.T) {
		// Given
		os.Setenv("test", "1")

		// When
		test, _ := defaultOrEnv("test", 2)

		// Assert
		assert.Equal(t, 1, test)
	})

	t.Run("defaultOrEnv returns default value if no env variable was set", func(t *testing.T) {
		// When
		os.Unsetenv("test")

		// When
		test, _ := defaultOrEnv("test", 2)

		// Assert
		assert.Equal(t, 2, test)
	})
}
