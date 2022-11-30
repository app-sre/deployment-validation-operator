package prometheus

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/stretchr/testify/assert"
)

// TestOptionsStruct runs four tests on options struct methods:
// -
func TestGetRouter(t *testing.T) {

	t.Run("router path is available", func(t *testing.T) {
		// Given
		recorder := httptest.NewRecorder()
		mockPath := "/test"
		mockReq, _ := http.NewRequest("GET", mockPath, nil)
		mockRegistry := prometheus.NewRegistry()

		// When
		mux, err := getRouter(mockRegistry, mockPath)
		mux.ServeHTTP(recorder, mockReq)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("handle error on collector registry", func(t *testing.T) {
		// Given
		mockRegistry := prometheus.NewRegistry()
		mockCollector := collectors.NewGoCollector()
		errMock := mockRegistry.Register(mockCollector)
		if errMock != nil {
			t.Errorf("Unexpected error at registering mock : %s", errMock.Error())
		}

		// When
		_, err := getRouter(mockRegistry, "/test")

		// Assert
		assert.Error(t, err)
		assert.IsType(t, registerCollectorError{}, err)
	})
}
