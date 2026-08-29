package httpserver

import (
	"net/http"
	"time"

	"github.com/prasannakumar414/profile-retrieval-service/http/controller"
	"github.com/prasannakumar414/profile-retrieval-service/http/helper"
)

type Config struct {
	Addr string
}

type Server struct {
	server *http.Server
}

func NewServer(config Config) *Server {
	if config.Addr == "" {
		config.Addr = ":8080"
	}

	healthController := controller.NewHealthController()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", helper.RequireMethod(http.MethodGet, healthController.Health))

	return &Server{
		server: &http.Server{
			Addr:              config.Addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}
