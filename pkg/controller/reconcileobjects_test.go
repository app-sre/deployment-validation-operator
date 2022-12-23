package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestGetPriorityVersion runs four tests on this method of resourceSet
// - checks existingVersion is returned if no groups are returned by the scheme
// - checks existingVersion is returned if there is no match on given group
// - checks existingVersion is returned if group version match existing version
// - checks currentVersion is returned if group version match current version
func TestGetPriorityVersion(t *testing.T) {

	t.Run("Empty PrioritizedVersionsAllGroups return", func(t *testing.T) {
		// Given
		mock := resourceSet{scheme: runtime.NewScheme()}

		// When
		test := mock.getPriorityVersion("", "existingVersion", "")

		// Assert
		assert.Equal(t, "existingVersion", test)
	})

	t.Run("No group match", func(t *testing.T) {
		// Given
		scheme := runtime.NewScheme()
		scheme.SetVersionPriority([]schema.GroupVersion{
			{Group: "group", Version: "version"},
		}...)
		mock := resourceSet{scheme: scheme}

		// When
		test := mock.getPriorityVersion("no-match", "existingVersion", "")

		// Assert
		assert.Equal(t, "existingVersion", test)
	})

	t.Run("Group match with existing version", func(t *testing.T) {
		// Given
		scheme := runtime.NewScheme()
		scheme.SetVersionPriority([]schema.GroupVersion{{Group: "group", Version: "existingVersion"}}...)
		scheme.SetVersionPriority([]schema.GroupVersion{{Group: "group2", Version: "currentVersion"}}...)
		mock := resourceSet{scheme: scheme}

		// When
		test := mock.getPriorityVersion("group", "existingVersion", "currentVersion")

		// Assert
		assert.Equal(t, "existingVersion", test)
	})

	t.Run("Group match with current version", func(t *testing.T) {
		// Given
		scheme := runtime.NewScheme()
		scheme.SetVersionPriority([]schema.GroupVersion{{Group: "group", Version: "existingVersion"}}...)
		scheme.SetVersionPriority([]schema.GroupVersion{{Group: "group2", Version: "currentVersion"}}...)
		mock := resourceSet{scheme: scheme}

		// When
		test := mock.getPriorityVersion("group2", "existingVersion", "currentVersion")

		// Assert
		assert.Equal(t, "currentVersion", test)
	})

	t.Run("getPriorityVersion : group match with no match version", func(t *testing.T) {
		// Given
		scheme := runtime.NewScheme()
		scheme.SetVersionPriority([]schema.GroupVersion{{Group: "group", Version: "version"}}...)
		scheme.SetVersionPriority([]schema.GroupVersion{{Group: "group2", Version: "version2"}}...)
		mock := resourceSet{scheme: scheme}

		// When
		test := mock.getPriorityVersion("group2", "existingVersion", "")

		// Assert
		assert.Equal(t, "existingVersion", test)
	})
}
