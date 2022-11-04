package testing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"
	dto "github.com/prometheus/client_model/go"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type TextParser interface {
	TextToMetricFamilies(in io.Reader) (map[string]*dto.MetricFamily, error)
}

func NewPromClient() *PromClient {
	var cfg PromClientConfig

	cfg.Default()

	return &PromClient{
		cfg: cfg,
	}
}

type PromClient struct {
	cfg PromClientConfig
}

var ErrFailedRequest = errors.New("failed request")

func (c *PromClient) GetDVOMetrics(ctx context.Context, url string) (map[string][]*io_prometheus_client.Metric, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating metrics request: %w", err)
	}

	res, err := c.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting metrics from %q: %w", url, err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, ErrFailedRequest
	}

	mfs, err := c.cfg.Parser.TextToMetricFamilies(res.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing metrics: %w", err)
	}

	metrics := make(map[string][]*io_prometheus_client.Metric)
	pfx := strings.ReplaceAll(config.OperatorName, "-", "_") + "_"

	for name, mf := range mfs {
		if strings.HasPrefix(name, pfx) {
			metrics[strings.TrimPrefix(name, pfx)] = mf.GetMetric()
		}
	}

	return metrics, nil
}

type PromClientConfig struct {
	Client *http.Client
	Parser TextParser
}

func (c *PromClientConfig) Default() {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}

	if c.Parser == nil {
		c.Parser = &expfmt.TextParser{}
	}
}
