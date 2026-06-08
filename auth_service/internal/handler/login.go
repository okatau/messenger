package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"auth_service/internal/domain"
	"auth_service/internal/service"

	"github.com/labstack/echo/v5"
)

func Login(auth service.Auth) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}

		addr, err := mail.ParseAddress(req.Email)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid email")
		}
		if req.Password == "" || len([]byte(req.Password)) > passwordMaxLen || len([]byte(req.Password)) < passwordMinLen {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid password")
		}

		user, err := auth.Login(c.Request().Context(), strings.ToLower(addr.Address), req.Password)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrUserForbidden):
				return echo.NewHTTPError(http.StatusUnauthorized, "user forbidden")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusOK, user)
	}
}
