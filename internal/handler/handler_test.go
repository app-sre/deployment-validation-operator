package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSwitchableHandler(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		HandlerA    http.Handler
		StatusCodeA int
		HandlerB    http.Handler
		StatusCodeB int
	}{
		"200/500": {
			HandlerA:    handleStatus(http.StatusOK),
			StatusCodeA: http.StatusOK,
			HandlerB:    handleStatus(http.StatusInternalServerError),
			StatusCodeB: http.StatusInternalServerError,
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := NewSwitchableHandler(tc.HandlerA, tc.HandlerB)

			assert.HTTPStatusCode(t, handler.ServeHTTP, http.MethodGet, "", nil, tc.StatusCodeA)

			handler.Switch()

			assert.HTTPStatusCode(t, handler.ServeHTTP, http.MethodGet, "", nil, tc.StatusCodeB)

			handler.Switch()

			assert.HTTPStatusCode(t, handler.ServeHTTP, http.MethodGet, "", nil, tc.StatusCodeA)
		})
	}
}

func handleStatus(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	})
}

func TestStopAfterNForwards(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		MaxForwards        uint
		ActualForwards     int
		ExpectedStatusCode int
	}{
		"max 3/actual 1": {
			MaxForwards:        3,
			ActualForwards:     1,
			ExpectedStatusCode: http.StatusOK,
		},
		"max 3/actual 5": {
			MaxForwards:        3,
			ActualForwards:     5,
			ExpectedStatusCode: http.StatusLoopDetected,
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := StopAfterNForwards(
				tc.MaxForwards, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			)

			w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://test", nil)

			clients := make([]string, 0, tc.ActualForwards)

			for i := 0; i < tc.ActualForwards; i++ {
				clients = append(clients, fmt.Sprint(i))
			}

			r.Header.Set("X-Forwarded-For", strings.Join(clients, ", "))

			handler.ServeHTTP(w, r)

			assert.Equal(t, tc.ExpectedStatusCode, w.Result().StatusCode)
		})
	}
}
