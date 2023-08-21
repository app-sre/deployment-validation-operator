package configmap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.stackrox.io/kube-linter/pkg/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestStaticConfigMapWatcher(t *testing.T) {
	testCases := []struct {
		name   string
		data   string
		checks config.ChecksConfig
	}{
		{
			name: "kube-linter 'doNotAutoAddDefaults' is gathered from configuration",
			data: "checks:\n  doNotAutoAddDefaults: true",
			checks: config.ChecksConfig{
				DoNotAutoAddDefaults: true,
			},
		},
		{
			name: "kube-linter 'addAllBuiltIn' is gathered from configuration",
			data: "checks:\n  addAllBuiltIn: true",
			checks: config.ChecksConfig{
				AddAllBuiltIn: true,
			},
		},
		{
			name: "kube-linter 'exclude' is gathered from configuration",
			data: "checks:\n  exclude: [\"check1\", \"check2\"]",
			checks: config.ChecksConfig{
				Exclude: []string{"check1", "check2"},
			},
		},
		{
			name: "kube-linter 'include' is gathered from configuration",
			data: "checks:\n  include: [\"check1\", \"check2\"]",
			checks: config.ChecksConfig{
				Include: []string{"check1", "check2"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Namespace: configMapNamespace, Name: configMapName},
				Data: map[string]string{
					"deployment-validation-operator-config.yaml": testCase.data,
				},
			}
			client := kubefake.NewSimpleClientset([]runtime.Object{cm}...)
			mock := Watcher{clientset: client}

			// When
			test, err := mock.GetStaticKubelinterConfig(context.Background())

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, testCase.checks, test.Checks)
		})
	}
}
