package server

import (
	"io"
	"net/http"
	"time"

	"github.com/0xPolygonID/refresh-service/logger"
	"github.com/0xPolygonID/refresh-service/service"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"github.com/rs/cors"
)

type Handlers struct {
	agentService *service.AgentService
}

func NewHandlers(
	agentService *service.AgentService,
) *Handlers {
	return &Handlers{
		agentService: agentService,
	}
}

func (h *Handlers) Run(host string) error {
	router := chi.NewRouter()
	// Basic CORS
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"localhost", "127.0.0.1", "*"},
		AllowedMethods: []string{"POST"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type",
			"X-CSRF-Token"},
		AllowCredentials: true,
	})
	router.Use(corsMiddleware.Handler)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(zapContextLogger)
	router.Use(middleware.Recoverer)

	router.Post("/", func(w http.ResponseWriter, r *http.Request) {
		envelope, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
		if err != nil {
			logger.DefaultLogger.Errorf("failed to read request body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response, err := h.agentService.Process(envelope)
		if err != nil {
			handleError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(response)
		if err != nil {
			logger.DefaultLogger.Errorf("failed to write response: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	logger.DefaultLogger.Infof("Server starting on host '%s'", host)
	httpServer := &http.Server{
		Addr:              host,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return errors.WithStack(httpServer.ListenAndServe())
}
