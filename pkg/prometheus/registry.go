package prometheus

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func NewRegistry() (*prometheus.Registry, error) {
	reg := prometheus.NewRegistry()

	processCollector := collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})
	if err := reg.Register(processCollector); err != nil {
		return nil, collectorRegistrationError("process", err)
	}

	goCollector := collectors.NewGoCollector()
	if err := reg.Register(goCollector); err != nil {
		return nil, collectorRegistrationError("go", err)
	}

	return reg, nil
}

func collectorRegistrationError(name string, err error) error {
	return fmt.Errorf("registering %s collector: %w", name, err)
}
