package handler

import (
	"errors"
	"friends_service/internal/domain"
	"friends_service/internal/service"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func SendFriendRequest(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)
		var req struct {
			InviteeID string `json:"inviteeId"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}

		if _, err := uuid.Parse(req.InviteeID); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid invitee id")
		}

		err := svc.SendFriendRequest(c.Request().Context(), userID, req.InviteeID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserInvalidInvitee):
				return echo.NewHTTPError(http.StatusNotFound, "invalid invitee id")
			case errors.Is(err, domain.ErrFriendReqAlreadyExists):
				return echo.NewHTTPError(http.StatusBadRequest, "friend request already exists")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusCreated, map[string]string{"message": "invite sent"})
	}
}
