package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"auth_service/internal/domain"
	"auth_service/internal/service"

	"github.com/labstack/echo/v5"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func Login(auth service.Auth) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}

		if !emailRegex.MatchString(req.Email) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid email")
		}
		if req.Password == "" || len([]byte(req.Password)) > 72 || len(req.Password) < 5 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid password")
		}

		req.Email = strings.ToLower(req.Email)

		userInfo, err := auth.Login(c.Request().Context(), req.Email, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrUserForbidden):
				return echo.NewHTTPError(http.StatusUnauthorized, "user forbidden")
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
