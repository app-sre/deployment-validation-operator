package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/app-sre/deployment-validation-operator/config"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"golang.stackrox.io/kube-linter/pkg/checkregistry"
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

// newPprofServeMux creates a new HTTP ServeMux with pprof handlers registered.
// It is intended for exposing pprof endpoints such as CPU and memory profiling.
func newPprofServeMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Register pprof handlers on the ServeMux with specific URL paths.
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
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

	mux := newPprofServeMux()
	mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return mux, nil
}

// PreloadMetrics preloads metrics related to predefined checks into the provided Prometheus registry.
// It retrieves predefined checks from the linter registry, sets up corresponding GaugeVec metrics,
// and registers them in the Prometheus registry.
//
// Parameters:
//   - pr: A pointer to a Prometheus registry where the metrics will be registered.
//
// Returns:
//   - A map of check names to corresponding GaugeVec metrics.
//   - An error if any error occurs during metric setup or registration.
func PreloadMetrics(pr *prometheus.Registry) (map[string]*prometheus.GaugeVec, error) {
	preloadedMetrics := make(map[string]*prometheus.GaugeVec)

	klr, err := validations.GetKubeLinterRegistry()
	if err != nil {
		return nil, err
	}

	checks, err := validations.GetAllNamesFromRegistry(klr)
	if err != nil {
		return nil, err
	}

	for _, checkName := range checks {
		metric, err := setupMetric(klr, checkName)
		if err != nil {
			return nil, fmt.Errorf("unable to create metric for check %s", checkName)
		}

		if err := pr.Register(metric); err != nil {
			return nil, fmt.Errorf("registering metric for check %q: %w", checkName, err)
		}

		preloadedMetrics[checkName] = metric
	}

	return preloadedMetrics, nil
}

// setupMetric sets up a Prometheus metric based on the provided checkname and information from a CheckRegistry.
// The metric is created with the formatted name, description, and remediation information from the check specification.
func setupMetric(reg checkregistry.CheckRegistry, name string) (*prometheus.GaugeVec, error) {
	check := reg.Load(name)
	if check == nil {
		return nil, fmt.Errorf("unable to create metric for check %s", name)
	}

	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: strings.ReplaceAll(
				fmt.Sprintf("%s_%s", config.OperatorName, check.Spec.Name),
				"-", "_"),
			Help: fmt.Sprintf(
				"Description: %s ; Remediation: %s",
				check.Spec.Description,
				check.Spec.Remediation,
			),
			ConstLabels: prometheus.Labels{
				"check_description": check.Spec.Description,
				"check_remediation": check.Spec.Remediation,
			},
		}, []string{"namespace_uid", "namespace", "uid", "name", "kind"}), nil
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
