package httpserver

import (
	"net/http"
	"time"

	"github.com/prasannakumar414/profile-retrieval-service/http/controller"
	"github.com/prasannakumar414/profile-retrieval-service/http/helper"
	"github.com/prasannakumar414/profile-retrieval-service/services"
)

type Config struct {
	Addr                       string
	ProfileRateLimitRequests   int
	ProfileRateLimitWindow     time.Duration
	LinkedInRequestMinInterval time.Duration
}

type Server struct {
	server *http.Server
}

func NewServer(config Config) *Server {
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.ProfileRateLimitRequests == 0 {
		config.ProfileRateLimitRequests = 1
	}
	if config.ProfileRateLimitWindow == 0 {
		config.ProfileRateLimitWindow = 2 * time.Minute
	}
	if config.LinkedInRequestMinInterval == 0 {
		config.LinkedInRequestMinInterval = services.DefaultLinkedInRequestMinInterval
	}

	healthController := controller.NewHealthController()
	docsController := controller.NewDocsController()
	profileService := services.NewProfileServiceWithConfig(services.ProfileServiceConfig{
		RequestMinInterval: config.LinkedInRequestMinInterval,
	})
	profileController := controller.NewProfileController(profileService)
	profileRetrieveHandler := profileController.Retrieve
	if config.ProfileRateLimitRequests > 0 && config.ProfileRateLimitWindow > 0 {
		profileRetrieveHandler = helper.NewRateLimiter(helper.RateLimitConfig{
			Requests: config.ProfileRateLimitRequests,
			Window:   config.ProfileRateLimitWindow,
		}).Wrap(profileRetrieveHandler)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", helper.RequireMethod(http.MethodGet, healthController.Health))
	mux.HandleFunc("/docs", helper.RequireMethod(http.MethodGet, docsController.Docs))
	mux.HandleFunc("/docs/", helper.RequireMethod(http.MethodGet, docsController.Docs))
	mux.HandleFunc("/docs/openapi.yaml", helper.RequireMethod(http.MethodGet, docsController.OpenAPI))
	mux.HandleFunc("/profiles/retrieve", profileRetrieveHandler)

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
