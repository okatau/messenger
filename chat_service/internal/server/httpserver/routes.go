package httpserver

import (
	"chat_service/internal/handler"
	"chat_service/internal/middleware"
	"chat_service/internal/service"
	"chat_service/pkg/token_manager"
	"context"

	"github.com/labstack/echo/v5"
)

func registreRoutes(router *echo.Echo, svc service.Hub) {
	authMW := middleware.ExtractUserID()

	router.GET("", handler.GetRoom(svc), authMW)
	router.GET("/:roomId/users", handler.GetUsersByRoom(svc), authMW)
	router.GET("/:roomId/messages", handler.GetRoomHistory(svc), authMW)
	router.POST("", handler.CreateRoom(svc), authMW)
	router.POST("/:roomId/invite", handler.InviteUser(svc), authMW)
	router.POST("/:roomId/leave", handler.LeaveRoom(svc), authMW)
}

func registerWS(ctx context.Context, router *echo.Echo, svc service.Hub, tm *token_manager.TokenManager, whitelist []string) {
	router.GET("/wss", handler.Connect(svc, tm, ctx, whitelist))
}
