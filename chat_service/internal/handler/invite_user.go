package handler

import (
	"errors"
	"net/http"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
)

func InviteUser(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		roomID := c.Param("roomId")
		//nolint:errcheck // userID sets in chat_service/internal/middleware/extract_userid.go
		inviterID := c.Get("userID").(string)
		var req struct {
			UserID string `json:"userId"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalud req body")
		}

		if roomID == "" || req.UserID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room id or user id")
		}

		err := hub.InviteUser(c.Request().Context(), inviterID, req.UserID, roomID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserForbidden):
				return echo.NewHTTPError(http.StatusForbidden, "forbidden")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.NoContent(http.StatusNoContent)
	}
}
