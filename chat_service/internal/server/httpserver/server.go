package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"chat_service/internal/service"
	"chat_service/pkg/config"
	"chat_service/pkg/service_logger"
	"chat_service/pkg/token_manager"
)

type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

func New(
	ctx context.Context,
	cfg config.ServerConfig,
	whitelist []string,
	svc service.Hub,
	logger *slog.Logger,
	tm *token_manager.TokenManager,
) *Server {
	router := echo.New()
	router.Use(service_logger.LoggerMW(logger))

	registreRoutes(router, svc) //nolint:contextcheck // no need extra context
	registerWS(ctx, router, svc, tm, whitelist)

	return &Server{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
			// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
			// WriteTimeout: cfg.ServerConfig.WriteTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info(fmt.Sprintf("listening chat service on %s", s.srv.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
