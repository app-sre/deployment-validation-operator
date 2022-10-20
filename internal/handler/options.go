package handler

import (
	"github.com/go-logr/logr"
)

type WithLog struct{ Log logr.Logger }

func (w WithLog) ConfigureSwitchableHandler(c *SwitchableHandlerConfig) {
	c.Log = w.Log
}
