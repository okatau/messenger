package handler

import (
	"errors"
	"net/http"

	"auth_service/internal/domain"
	"auth_service/internal/service"

	"github.com/labstack/echo/v5"
)

func Refresh(auth service.Auth) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}
		if req.RefreshToken == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid refresh token")
		}

		user, err := auth.Refresh(c.Request().Context(), req.RefreshToken)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrTokenNotFound):
				return echo.NewHTTPError(http.StatusUnauthorized, "token not found")
			case errors.Is(err, domain.ErrTokenExpired):
				return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusOK, user)
	}
}
