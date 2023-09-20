package configmap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.stackrox.io/kube-linter/pkg/config"
)

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name           string
		configData     string
		expectedConfig config.Config
		expectedError  error
	}{
		{
			name: "Basic valid config",
			configData: `
checks:
  doNotAutoAddDefaults: false
  addAllBuiltIn: true
  include:
  - "unset-memory-requirements"
  - "unset-cpu-requirements"`,
			expectedConfig: config.Config{
				Checks: config.ChecksConfig{
					AddAllBuiltIn:        true,
					DoNotAutoAddDefaults: false,
					Include:              []string{"unset-memory-requirements", "unset-cpu-requirements"}, // nolint: lll
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid config field \"doNotAutoAddDefaultsAAA\"",
			configData: `
checks:
  doNotAutoAddDefaultsAAA: false
  addAllBuiltIn: true
  include:
  - "unset-memory-requirements"
  - "unset-cpu-requirements"`,
			expectedError: fmt.Errorf("unmarshalling configmap data: error unmarshaling JSON: while decoding JSON: json: unknown field \"doNotAutoAddDefaultsAAA\""), // nolint: lll
			expectedConfig: config.Config{
				Checks: config.ChecksConfig{
					AddAllBuiltIn:        true,
					DoNotAutoAddDefaults: false,
					Include:              []string{"unset-memory-requirements", "unset-cpu-requirements"}, // nolint: lll
				},
			},
		},
		{
			name: "Invalid config field \"include\"",
			configData: `
checks:
  doNotAutoAddDefaults: false
  addAllBuiltIn: true
  includeaa:
  - "unset-memory-requirements"
  - "unset-cpu-requirements"`,
			expectedError: fmt.Errorf("unmarshalling configmap data: error unmarshaling JSON: while decoding JSON: json: unknown field \"includeaa\""), // nolint: lll
			expectedConfig: config.Config{
				Checks: config.ChecksConfig{
					AddAllBuiltIn:        true,
					DoNotAutoAddDefaults: false,
				},
			},
		},
		{
			name: "Valid config with custom check",
			configData: `
checks:
  doNotAutoAddDefaults: false
  addAllBuiltIn: true
  include:
  - "unset-memory-requirements"
customChecks:
  - name: test-minimum-replicas
    description: "some description"
    remediation: "some remediation"
    template: minimum-replicas
    params:
      minReplicas: 3
    scope:
      objectKinds:
        - DeploymentLike`,
			expectedError: nil,
			expectedConfig: config.Config{
				Checks: config.ChecksConfig{
					AddAllBuiltIn:        true,
					DoNotAutoAddDefaults: false,
					Include:              []string{"unset-memory-requirements"},
				},
				CustomChecks: []config.Check{
					{
						Name:        "test-minimum-replicas",
						Description: "some description",
						Remediation: "some remediation",
						Template:    "minimum-replicas",
						Params: map[string]interface{}{
							"minReplicas": float64(3),
						},
						Scope: &config.ObjectKindsDesc{
							ObjectKinds: []string{"DeploymentLike"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			cfg, err := readConfig(tt.configData)
			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedConfig, cfg)
		})
	}
}
