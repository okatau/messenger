package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"friends_service/internal/domain"
	"friends_service/internal/service"
)

func SendFriendRequest(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in friends_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		var req struct {
			InviteeID string `json:"inviteeId"`
		}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}

		if _, err := uuid.Parse(req.InviteeID); err != nil || userID == req.InviteeID {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid invitee id")
		}

		err := svc.SendFriendRequest(c.Request().Context(), userID, req.InviteeID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserNotFound):
				return echo.NewHTTPError(http.StatusNotFound, "invalid invitee id")
			case errors.Is(err, domain.ErrRequestAlreadyExists):
				return echo.NewHTTPError(http.StatusBadRequest, "friend request already exists")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.NoContent(http.StatusCreated)
	}
}
