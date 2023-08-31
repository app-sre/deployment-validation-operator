package validations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.stackrox.io/kube-linter/pkg/config"
)

func TestGetValidChecks(t *testing.T) {
	testCases := []struct {
		name     string
		included []string
		expected []string
	}{
		{
			name:     "it returns only included checks",
			included: []string{"host-network", "host-pid"},
			expected: []string{"host-network", "host-pid"},
		},
		{
			name:     "it returns only validated checks, and not the misspelled",
			included: []string{"host-network", "host-pid", "misspelled", "wrong_format"},
			expected: []string{"host-network", "host-pid"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			mock := ValidationEngine{
				config: config.Config{
					Checks: config.ChecksConfig{
						DoNotAutoAddDefaults: true,
						Include:              testCase.included,
					},
				},
			}
			registry, _ := GetKubeLinterRegistry()

			// When
			test, err := mock.getValidChecks(registry)

			// Assert
			assert.Equal(t, testCase.expected, test)
			assert.NoError(t, err)
		})
	}
}

func TestRemoveCheckFromConfig(t *testing.T) {
	testCases := []struct {
		name     string
		check    string
		cfg      config.ChecksConfig
		expected config.ChecksConfig
	}{
		{
			name:  "function removes misspelled check from Include list",
			check: "unset-something",
			cfg: config.ChecksConfig{
				Include: []string{"host-network", "host-pid", "unset-something"},
			},
			expected: config.ChecksConfig{
				Include: []string{"host-network", "host-pid"},
			},
		},
		{
			name:  "function removes misspelled check from Exclude list",
			check: "unset-something",
			cfg: config.ChecksConfig{
				Exclude: []string{"host-network", "host-pid", "unset-something"},
			},
			expected: config.ChecksConfig{
				Exclude: []string{"host-network", "host-pid"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			mock := ValidationEngine{config: config.Config{Checks: testCase.cfg}}

			// When
			mock.removeCheckFromConfig(testCase.check)

			// Assert
			assert.Equal(t, mock.config.Checks, testCase.expected)
		})
	}
}
