package prometheus

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/app-sre/deployment-validation-operator/internal/runnable"
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
		Server: runnable.NewHTTPServer(mux, addr),
	}, nil
}

type Server struct {
	runnable.Server
}
