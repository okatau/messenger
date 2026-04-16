package handler

import (
	"auth_service/internal/domain"
	"auth_service/internal/service"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

func Register(auth service.Auth) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req struct {
			Username string `json:"username"`
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
		if req.Username == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
		}

		req.Username = strings.ToLower(req.Username)
		req.Email = strings.ToLower(req.Email)

		user, err := auth.Register(c.Request().Context(), req.Username, req.Email, req.Password)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserExist):
				echo.NewHTTPError(http.StatusUnauthorized, "user forbidden")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "server internal error")
			}
		}

		return c.JSON(http.StatusCreated, map[string]any{
			"user_id":  user.ID,
			"email":    user.Email,
			"username": user.Name,
		})
	}
}
