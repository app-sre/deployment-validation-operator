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

// registerCollectorError is returned by the NewServer method if the Registry
// causes any error on registering a Collector
type registerCollectorError struct {
	collector string
	trace     error
}

func (err registerCollectorError) Error() string {
	return fmt.Sprintf("registering %s collector: %s", err.collector, err.trace.Error())
}

// NewServer returns a server configured to work on path and address given
// It registers a collector which exports the current state of process metrics
// and a collector that exports metrics about the current process
// It may return registerCollectorError if the collectors are already registered
func NewServer(registry Registry, path, addr string) (*Server, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	mux, err := getRouter(registry, path)
	if err != nil {
		return nil, err
	}

	return &Server{
		s: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 2 * time.Second,
		},
	}, nil
}

// getRouter registers two collectors to deliver metrics on a given path
// It may return registerCollectorError if the collectors are already registered
func getRouter(registry Registry, path string) (*http.ServeMux, error) {
	var (
		processCollector = collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
		goCollector      = collectors.NewGoCollector()
	)

	if err := registry.Register(processCollector); err != nil {
		return nil, registerCollectorError{
			collector: "process",
			trace:     err,
		}
	}

	if err := registry.Register(goCollector); err != nil {
		return nil, registerCollectorError{
			collector: "go",
			trace:     err,
		}
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return mux, nil
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
