package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func ExtractUserID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			userID := c.Request().Header.Get("X-User-ID")
			if userID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing user id")
			}
			c.Set("userID", userID)
			return next(c)
		}
	}
}
