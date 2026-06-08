package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"auth_service/internal/components"
	"auth_service/pkg/config"
	sl "auth_service/pkg/service_logger"
)

type Server struct {
	srv   *http.Server
	comps *components.Components
}

func New() *Server {
	ctx := context.Background()
	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()
	comps := components.InitComponents(ctxTimeout, cfg)

	router := echo.New()
	router.Use(sl.LoggerMW(comps.Logger))
	registreRoutes(router, comps.Svc)

	return &Server{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
			Handler: router,
			// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
			// WriteTimeout: cfg.ServerConfig.WriteTimeout,
		},
		comps: comps,
	}
}

func (s *Server) Start() error {
	s.comps.Logger.Info(fmt.Sprintf("listening auth service on %s", s.srv.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.comps.Logger.Error("error shutting down server")
	}
	s.comps.Shutdown()
}
