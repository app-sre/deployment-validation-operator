package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestConfigMapWatcher(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "testestest"},
		Data: map[string]string{
			"disabled-checks": "blablabla",
		},
	}

	// Given
	client := kubefake.NewSimpleClientset([]runtime.Object{cm}...).CoreV1()
	mock := configMapWatcher{}

	// When
	err := mock.withoutInformer(client)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, []string{"blablabla"}, mock.GetDisabledChecks())
}
