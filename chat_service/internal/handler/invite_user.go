package handler

import (
	"errors"
	"net/http"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
)

// TODO
// Now always force intvite even if user dont want join room
// Update logic to:
// 1. User sets availability of invites (alter table invite_available)
// 2. make accept / decline logic. create table invites (inviter, invitee, room, created at) and show to user when he is online
func InviteUser(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		roomID := c.Param("roomId")
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

		return c.JSON(http.StatusNoContent, nil)
	}
}
