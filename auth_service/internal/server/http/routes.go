package httpserver

import (
	"auth_service/internal/handler"
	"auth_service/internal/service"

	"github.com/labstack/echo/v5"
)

func registreRoutes(router *echo.Echo, svc service.Auth) {
	router.POST("/register", handler.Register(svc))
	router.POST("/login", handler.Login(svc))
	router.POST("/refresh", handler.Refresh(svc))
	router.POST("/logout", handler.Logout(svc))
}
