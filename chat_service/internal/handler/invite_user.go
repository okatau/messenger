package handler

import (
	"errors"
	"net/http"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func InviteUser(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in chat_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		roomID := c.Param("roomId")
		rID, err := uuid.Parse(roomID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room id")
		}

		var req struct {
			Invitee string `json:"inviteeId"`
		}
		if err = c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}

		if _, err = uuid.Parse(req.Invitee); err != nil || req.Invitee == userID {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid invitee id")
		}

		err = hub.InviteUser(c.Request().Context(), userID, req.Invitee, rID.String())
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserForbidden):
				return echo.NewHTTPError(http.StatusForbidden, "user forbidden")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.NoContent(http.StatusOK)
	}
}
