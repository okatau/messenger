package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/service"

	"github.com/labstack/echo/v5"
)

const (
	passwordMaxLen = 72
	passwordMinLen = 5

	usernameMaxLen = 72
	usernameMinLen = 5
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

		addr, err := mail.ParseAddress(req.Email)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid email")
		}
		if req.Password == "" || len([]byte(req.Password)) > passwordMaxLen || len([]byte(req.Password)) < passwordMinLen {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid password")
		}
		if req.Username == "" || len([]byte(req.Username)) < passwordMinLen || len([]byte(req.Username)) > passwordMaxLen {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
		}

		user, err := auth.Register(
			c.Request().Context(),
			strings.ToLower(req.Username),
			strings.ToLower(addr.Address),
			req.Password,
		)

		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserExists):
				return echo.NewHTTPError(http.StatusConflict, "user already exists")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		type response struct {
			ID        string    `json:"userId"`
			Username  string    `json:"username"`
			Email     string    `json:"email"`
			CreatedAt time.Time `json:"createdAt"`
		}

		return c.JSON(
			http.StatusCreated,
			response{
				ID:        user.ID,
				Email:     user.Email,
				Username:  user.Username,
				CreatedAt: user.CreatedAt,
			},
		)
	}
}
