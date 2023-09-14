package validations

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/prometheus/client_golang/prometheus"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRequest(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		Object         client.Object
		NamespaceUID   string
		ExpectedLabels prometheus.Labels
	}{
		"without namespace UID": {
			Object: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
					UID:       "abcdefgh",
				},
			},
			ExpectedLabels: prometheus.Labels{
				"kind":          "Deployment",
				"name":          "test",
				"namespace":     "test-namespace",
				"namespace_uid": "",
				"uid":           "abcdefgh",
			},
		},
		"with namespace UID": {
			Object: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
					UID:       "abcdefgh",
				},
			},
			NamespaceUID: "12345678",
			ExpectedLabels: prometheus.Labels{
				"kind":          "Deployment",
				"name":          "test",
				"namespace":     "test-namespace",
				"namespace_uid": "12345678",
				"uid":           "abcdefgh",
			},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := NewRequestFromObject(tc.Object)
			req.NamespaceUID = tc.NamespaceUID

			assert.Equal(t, tc.ExpectedLabels, req.ToPromLabels())
		})
	}
}
