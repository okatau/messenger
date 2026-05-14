package handlers

import (
	"api_gateway/internal/components"
	"log"
	"net/http/httputil"

	"github.com/labstack/echo/v5"
)

func redirectTo(proxy *httputil.ReverseProxy) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if userID, ok := c.Get("userID").(string); ok && userID != "" {
			c.Request().Header.Set("X-User-ID", userID)
		}
		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

func InitAuthEndpoints(
	auth *echo.Group,
	targetUrl string,
	al components.AuthLimits,
	rl func(limit int) echo.MiddlewareFunc,
) {
	proxy, err := createProxy(targetUrl, "/api/v1/auth")
	if err != nil {
		log.Fatal(err)
	}

	auth.POST("/register", redirectTo(proxy), rl(al.RegisterLimit))
	auth.POST("/login", redirectTo(proxy), rl(al.LoginLimit))
	auth.POST("/refresh", redirectTo(proxy))
	auth.POST("/logout", redirectTo(proxy))
}
