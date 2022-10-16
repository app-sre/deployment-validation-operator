package runnable

import (
	"context"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Server interface {
	manager.Runnable
}

func NewHTTPServer(handler http.Handler, addr string) *HTTPServer {
	return &HTTPServer{
		s: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
}

type HTTPServer struct {
	s *http.Server
}

func (s *HTTPServer) Start(ctx context.Context) error {
	errCh := make(chan error)
	drain := func() {
		for range errCh {
		}
	}

	defer drain()

	go func() {
		defer close(errCh)

		errCh <- s.s.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.s.Close()
	}
}
