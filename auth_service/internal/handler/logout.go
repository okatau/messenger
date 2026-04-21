package handler

import (
	"errors"
	"net/http"

	"auth_service/internal/domain"
	"auth_service/internal/service"

	"github.com/labstack/echo/v5"
)

func Logout(auth service.Auth) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}

		if req.RefreshToken == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid refresh token")
		}

		err := auth.Logout(c.Request().Context(), req.RefreshToken)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrTokenNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "server internal error")
			}
		}

		return c.NoContent(http.StatusNoContent)
	}
}
