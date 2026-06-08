package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func ExtractUserID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			userID := c.Request().Header.Get("X-User-ID")
			uID, err := uuid.Parse(userID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing user id")
			}
			c.Set("userID", uID.String())
			return next(c)
		}
	}
}
