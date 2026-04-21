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

func Test_Handler_GetFriendsList(t *testing.T) {
	friends := []*domain.User{{ID: bobID, Username: "bob"}}

	tests := []struct {
		name       string
		setup      func(svc *friendshipSvcMock)
		wantStatus int
	}{
		{
			name: "success",
			setup: func(svc *friendshipSvcMock) {
				svc.On("GetFriendsList", mock.Anything, aliceID).Return(friends, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "internal error",
			setup: func(svc *friendshipSvcMock) {
				svc.On("GetFriendsList", mock.Anything, aliceID).Return([]*domain.User{}, dbError)
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

			c, rec := newContext(http.MethodGet, "/friends", "")
			c.Set("userID", aliceID)

			err := GetFriendsList(svc)(c)

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
