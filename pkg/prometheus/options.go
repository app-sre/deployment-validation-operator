package prometheus

import (
	"github.com/go-logr/logr"
)

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureServer(c *ServerConfig) {
	c.Log = w.Log
}

type WithMaxForwardAttempts uint

func (w WithMaxForwardAttempts) ConfigureServer(c *ServerConfig) {
	c.MaxForwardAttempts = uint(w)
}

type WithMetricsAddr string

func (w WithMetricsAddr) ConfigureServer(c *ServerConfig) {
	c.MetricsAddr = string(w)
}

type WithMetricsPath string

func (w WithMetricsPath) ConfigureServer(c *ServerConfig) {
	c.MetricsPath = string(w)
}

type WithServiceURL string

func (w WithServiceURL) ConfigureServer(c *ServerConfig) {
	c.ServiceURL = string(w)
}
