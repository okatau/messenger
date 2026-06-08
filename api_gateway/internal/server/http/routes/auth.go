package routes

import (
	"net/http/httputil"

	"github.com/labstack/echo/v5"

	"api_gateway/internal/components"
)

func InitAuthEndpoints(
	proxy *httputil.ReverseProxy,
	auth *echo.Group,
	limitFn func(limit int) echo.MiddlewareFunc,
	limits components.AuthLimits,
	redirect func(proxy *httputil.ReverseProxy) echo.HandlerFunc,
) {
	auth.POST("/register", redirect(proxy), limitFn(limits.RegisterLimit))
	auth.POST("/login", redirect(proxy), limitFn(limits.LoginLimit))
	auth.POST("/refresh", redirect(proxy))
	auth.POST("/logout", redirect(proxy))
}
