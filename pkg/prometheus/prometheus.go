package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

type Registry interface {
	Register(prometheus.Collector) error
	Gather() ([]*dto.MetricFamily, error)
}

func NewServer(registry Registry, path, addr string) (*Server, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var (
		processCollector = collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
		goCollector      = collectors.NewGoCollector()
	)

	if err := registry.Register(processCollector); err != nil {
		return nil, fmt.Errorf("registering process collector: %w", err)
	}

	if err := registry.Register(goCollector); err != nil {
		return nil, fmt.Errorf("registering go collector: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return &Server{
		s: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}, nil
}

type Server struct {
	s *http.Server
}

func (s *Server) Start(ctx context.Context) error {
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
