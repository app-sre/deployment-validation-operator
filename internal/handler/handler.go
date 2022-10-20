package handler

import (
	"net/http"
	"strings"
	"sync"

	"github.com/go-logr/logr"
)

func NewSwitchableHandler(handA http.Handler, handB http.Handler, opts ...SwitchableHandlerOption) *SwitchableHandler {
	var cfg SwitchableHandlerConfig

	cfg.Option(opts...)
	cfg.Default()

	return &SwitchableHandler{
		cfg:      cfg,
		handlerA: handA,
		handlerB: handB,
	}
}

type SwitchableHandler struct {
	cfg      SwitchableHandlerConfig
	handlerA http.Handler
	handlerB http.Handler

	lock     sync.Mutex
	switched bool
}

func (h *SwitchableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.cfg.Log.WithValues("remote", r.RemoteAddr)

	if h.switched {
		log.WithValues("handler", "B").Info("serving request")

		h.handlerB.ServeHTTP(w, r)
	} else {
		log.WithValues("handler", "A").Info("serving request")

		h.handlerA.ServeHTTP(w, r)
	}
}

func (h *SwitchableHandler) Switch() {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.switched {
		h.switched = false
	} else {
		h.switched = true
	}
}

type SwitchableHandlerConfig struct {
	Log logr.Logger
}

func (c *SwitchableHandlerConfig) Option(opts ...SwitchableHandlerOption) {
	for _, opt := range opts {
		opt.ConfigureSwitchableHandler(c)
	}
}

func (c *SwitchableHandlerConfig) Default() {
	if c.Log == nil {
		c.Log = logr.Discard()
	}
}

type SwitchableHandlerOption interface {
	ConfigureSwitchableHandler(*SwitchableHandlerConfig)
}

func StopAfterNForwards(n uint, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawFwds, ok := r.Header["X-Forwarded-For"]
		if !ok {
			h.ServeHTTP(w, r)
		}

		splitFwds := strings.Split(strings.Join(rawFwds, ", "), ", ")
		if uint(len(splitFwds)) >= n {
			http.Error(w, "", http.StatusLoopDetected)
		}

		h.ServeHTTP(w, r)
	})
}
