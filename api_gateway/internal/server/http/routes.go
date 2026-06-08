package httpserver

import (
	"api_gateway/internal/components"
	"api_gateway/internal/server/http/routes"

	"github.com/labstack/echo/v5"
)

func registreAuthRoutes(
	group *echo.Group,
	targetURL string,
	limitFn func(int) echo.MiddlewareFunc,
	limits components.AuthLimits,

) error {
	proxy, err := NewProxy(targetURL, "/api/v1/auth")
	if err != nil {
		return err
	}

	auth := group.Group("/auth")
	routes.InitAuthEndpoints(
		proxy,
		auth,
		limitFn,
		limits,
		RedirectTo,
	)

	return nil
}

func registerChatRoutes(
	group *echo.Group,
	targetURL string,
	limitFn func(int) echo.MiddlewareFunc,
	limits components.ChatLimits,
	authMW echo.MiddlewareFunc,
) error {
	proxy, err := NewProxy(targetURL, "/api/v1/rooms")
	if err != nil {
		return err
	}

	rooms := group.Group("/rooms")
	routes.InitChatEndpoints(
		proxy,
		rooms,
		limitFn,
		limits,
		RedirectTo,
		authMW,
	)

	return nil
}

func registerFriendsRoutes(
	group *echo.Group,
	targetURL string,
	limitFn func(int) echo.MiddlewareFunc,
	limits components.FriendsLimits,
	authMW echo.MiddlewareFunc,
) error {
	proxy, err := NewProxy(targetURL, "/api/v1/friends")
	if err != nil {
		return err
	}

	friends := group.Group("/friends")
	routes.InitFriendsEndpoints(
		proxy,
		friends,
		limitFn,
		limits,
		RedirectTo,
		authMW,
	)

	return nil
}
