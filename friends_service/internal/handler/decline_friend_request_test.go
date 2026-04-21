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

func Test_Handler_DeclineFriendRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(svc *friendshipSvcMock)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"inviterId":"` + bobID + `"}`,
			setup: func(svc *friendshipSvcMock) {
				svc.On("DeclineFriendRequest", mock.Anything, aliceID, bobID).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json body",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid uuid",
			body:       `{"inviterId":"not-a-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "request not found",
			body: `{"inviterId":"` + bobID + `"}`,
			setup: func(svc *friendshipSvcMock) {
				svc.On("DeclineFriendRequest", mock.Anything, aliceID, bobID).Return(domain.ErrFriendReqNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "internal error",
			body: `{"inviterId":"` + bobID + `"}`,
			setup: func(svc *friendshipSvcMock) {
				svc.On("DeclineFriendRequest", mock.Anything, aliceID, bobID).Return(dbError)
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

			c, rec := newContext(http.MethodPost, "/friends/requests/decline", tt.body)
			c.Set("userID", aliceID)

			err := DeclineFriendRequest(svc)(c)

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
