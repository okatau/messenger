package handler

import (
	"errors"
	"friends_service/internal/domain"
	"friends_service/internal/service"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func AcceptFriendRequest(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)
		var req struct {
			InviterID string `json:"inviterId"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}

		if _, err := uuid.Parse(req.InviterID); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid inviter id")
		}
		err := svc.AcceptFriendRequest(c.Request().Context(), userID, req.InviterID)

		if err != nil {
			switch {
			case errors.Is(err, domain.ErrFriendReqNotFound):
				return echo.NewHTTPError(http.StatusNotFound, "friend request not found")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusOK, map[string]string{"message": "invite accepted"})
	}
}
