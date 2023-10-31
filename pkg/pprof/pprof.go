package pprof

import (
	"context"
	"net/http"
	"time"
)

func NewServer() *Pprof {
	return &Pprof{
		s: &http.Server{
			Addr:              "localhost:6060",
			ReadHeaderTimeout: 2 * time.Second,
		},
	}
}

type Pprof struct {
	s *http.Server
}

func (s *Pprof) Start(ctx context.Context) error {
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
