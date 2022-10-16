package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/app-sre/deployment-validation-operator/internal/runnable"
)

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return &Server{
		Server: runnable.NewHTTPServer(mux, addr),
	}
}

type Server struct {
	runnable.Server
}
