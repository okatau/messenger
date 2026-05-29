package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"chat_service/internal/service"
)

func CreateDM(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in chat_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		var req struct {
			InviteeID string `json:"inviteeId"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}
		if req.InviteeID == "" || req.InviteeID == userID {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid invitee id")
		}

		room, err := hub.CreateDM(c.Request().Context(), userID, req.InviteeID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusCreated, room)
	}
}
