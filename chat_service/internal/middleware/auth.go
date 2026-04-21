package middleware

import (
	"net/http"
	"strings"

	"chat_service/pkg/token_manager"

	"github.com/labstack/echo/v5"
)

func Auth(manager *token_manager.TokenManager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header")
			}

			tokenStr := parts[1]
			claims, err := manager.VerifyAccessToken(tokenStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization token")
			}

			c.Set("userID", claims.Subject)
			return next(c)
		}
	}
}
