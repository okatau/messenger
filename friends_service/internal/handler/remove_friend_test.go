package handler

import (
	"errors"
	"friends_service/internal/domain"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Handler_RemoveFriend(t *testing.T) {
	tests := []struct {
		name       string
		friendID   string
		setup      func(svc *friendshipSvcMock)
		wantStatus int
	}{
		{
			name:     "success",
			friendID: bobID,
			setup: func(svc *friendshipSvcMock) {
				svc.On("RemoveFriend", mock.Anything, aliceID, bobID).Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid uuid",
			friendID:   "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "friend not found",
			friendID: bobID,
			setup: func(svc *friendshipSvcMock) {
				svc.On("RemoveFriend", mock.Anything, aliceID, bobID).Return(domain.ErrFriendNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "internal error",
			friendID: bobID,
			setup: func(svc *friendshipSvcMock) {
				svc.On("RemoveFriend", mock.Anything, aliceID, bobID).Return(dbError)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &friendshipSvcMock{}
			if tt.setup != nil {
				tt.setup(svc)
			}

			c, rec := newContext(http.MethodDelete, "/friends/"+tt.friendID, "")
			c.Set("userID", aliceID)
			c.SetPathValues(echo.PathValues{{Name: "friendId", Value: tt.friendID}})

			err := RemoveFriend(svc)(c)

			var httpErr *echo.HTTPError
			if errors.As(err, &httpErr) {
				require.Equal(t, tt.wantStatus, httpErr.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}

			svc.AssertExpectations(t)
		})
	}
}
