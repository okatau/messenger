package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"chat_service/internal/domain"
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
		inviteeID, err := uuid.Parse(req.InviteeID)
		if err != nil || inviteeID.String() == userID {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid invitee id")
		}

		room, err := hub.CreateDM(c.Request().Context(), userID, inviteeID.String())
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserForbidden):
				return echo.NewHTTPError(http.StatusForbidden, "user forbidden")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusCreated, room)
	}
}
