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
			RefreshToken string `json:"refresh_token"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}
		if req.RefreshToken == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid refresh token")
		}

		userInfo, err := auth.Refresh(c.Request().Context(), req.RefreshToken)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrTokenNotFound):
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			case errors.Is(err, domain.ErrTokenExpired):
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "server internal error")
			}
		}

		return c.JSON(http.StatusOK, map[string]any{
			"username":      userInfo.Username,
			"user_id":       userInfo.UserID,
			"access_token":  userInfo.AccessToken,
			"refresh_token": userInfo.RefreshToken,
		})
	}
}
