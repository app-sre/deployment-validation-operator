package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"net/http/httputil"

	"github.com/app-sre/deployment-validation-operator/internal/handler"
	"github.com/app-sre/deployment-validation-operator/internal/runnable"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewServer(gatherer prometheus.Gatherer, opts ...ServerOption) (*Server, error) {
	var cfg ServerConfig

	cfg.Option(opts...)
	cfg.Default()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation options: %w", err)
	}

	path := cfg.MetricsPath
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url, err := url.Parse(cfg.ServiceURL)
	if err != nil {
		return nil, fmt.Errorf("parsing service URL: %w", err)
	}

	h := handler.NewSwitchableHandler(
		handler.StopAfterNForwards(cfg.MaxForwardAttempts, httputil.NewSingleHostReverseProxy(url)),
		promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}),
		handler.WithLog{Log: cfg.Log},
	)

	mux := http.NewServeMux()
	mux.Handle(path, h)

	return &Server{
		s:         runnable.NewHTTPServer(mux, cfg.MetricsAddr),
		readyFunc: h.Switch,
	}, nil
}

func (s *Server) NeedLeaderElection() bool { return false }

type Server struct {
	s         runnable.Server
	readyFunc func()
}

func (s *Server) Ready() {
	s.readyFunc()
}

func (s *Server) Start(ctx context.Context) error {
	return s.s.Start(ctx)
}

type ServerConfig struct {
	Log                logr.Logger
	MaxForwardAttempts uint
	MetricsAddr        string
	MetricsPath        string
	ServiceURL         string
}

func (c *ServerConfig) Option(opts ...ServerOption) {
	for _, opt := range opts {
		opt.ConfigureServer(c)
	}
}

func (c *ServerConfig) Default() {
	if c.Log == nil {
		c.Log = logr.Discard()
	}

	if c.MaxForwardAttempts == 0 {
		c.MaxForwardAttempts = 10
	}
}

var ErrEmptyValue = errors.New("empty value")

func (c *ServerConfig) Validate() error {
	if c.MetricsAddr == "" {
		return fmt.Errorf("validating metrics address: %w", ErrEmptyValue)
	}

	if c.MetricsPath == "" {
		return fmt.Errorf("validating metrics path: %w", ErrEmptyValue)
	}

	if c.ServiceURL == "" {
		return fmt.Errorf("validating service URL: %w", ErrEmptyValue)
	}

	return nil
}

type ServerOption interface {
	ConfigureServer(c *ServerConfig)
}
