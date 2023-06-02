package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestStaticConfigMapWatcher(t *testing.T) {
	// Given
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: configMapNamespace, Name: configMapName},
		Data:       map[string]string{"disabled-checks": "check,check2"},
	}
	client := kubefake.NewSimpleClientset([]runtime.Object{cm}...)
	mock := ConfigMapWatcher{
		clientset: client,
	}

	// When
	test, err := mock.GetStaticKubelinterConfig(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, []string{"check", "check2"}, test)
}
