package controller

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestWatchNamespacesCache runs five tests on watchNamespacesCache struct methods:
// - checks that instantiation takes care of env variable conf
// - checks getNamespaceUID returns valid uid or empty string depending on argument
// - checks getFormattedNamespaces returns formatted
// - checks getFormattedNamespaces filters unwanted namespaces
func TestWatchNamespacesCache(t *testing.T) {

	t.Run("instantiation with ignore pattern Env variable set", func(t *testing.T) {
		// Given
		os.Setenv(EnvNamespaceIgnorePattern, "mock")

		// When
		wnc := newWatchNamespacesCache()

		// Assert
		assert.Equal(t, "mock", wnc.ignorePattern.String())
	})

	t.Run("getNamespaceUID returns an existing uid", func(t *testing.T) {
		// Given
		expected := "mock"
		wnc := watchNamespacesCache{
			namespaces: &[]namespace{
				{name: "test", uid: expected},
			},
		}

		// When
		test := wnc.getNamespaceUID("test")

		// Assert
		assert.Equal(t, expected, test)
	})

	t.Run("getNamespaceUID returns void string on non-existing uid", func(t *testing.T) {
		// Given
		wnc := watchNamespacesCache{
			namespaces: &[]namespace{
				{name: "test", uid: "mock"},
			},
		}

		// When
		test := wnc.getNamespaceUID("fail")

		// Assert
		assert.Equal(t, "", test)
	})

	t.Run("getFormattedNamespaces returns object data formatted", func(t *testing.T) {
		// Given
		expectedName := "mock"
		expectedUID := "1234"
		mockNamespaceList := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: expectedName, UID: types.UID(expectedUID),
					},
				},
			},
		}

		// When
		ns := getFormattedNamespaces(mockNamespaceList, nil)

		// Assert
		assert.Len(t, ns, 1)
		assert.Equal(t, expectedName, ns[0].name)
		assert.Equal(t, expectedUID, ns[0].uid)
	})

	t.Run("getFormattedNamespaces ignores namespace with given pattern", func(t *testing.T) {
		// Given
		mockNamespaceList := corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: v1.ObjectMeta{Name: "mock"},
				},
			},
		}

		// When
		ns := getFormattedNamespaces(mockNamespaceList, regexp.MustCompile("mock"))

		// Assert
		assert.Len(t, ns, 0)
	})
}
