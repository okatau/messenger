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

func Test_Handler_SearchUser(t *testing.T) {
	users := []*domain.User{{ID: bobID, Username: "bob"}}

	tests := []struct {
		name       string
		query      string
		setup      func(svc *friendshipSvcMock)
		wantStatus int
	}{
		{
			name:  "success",
			query: "?username=bob",
			setup: func(svc *friendshipSvcMock) {
				svc.On("FindMatchingUsers", mock.Anything, "bob", "").Return(users, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:  "with cursor",
			query: "?username=bob&cursor=abc",
			setup: func(svc *friendshipSvcMock) {
				svc.On("FindMatchingUsers", mock.Anything, "bob", "abc").Return(users, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing username",
			query:      "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "internal error",
			query: "?username=bob",
			setup: func(svc *friendshipSvcMock) {
				svc.On("FindMatchingUsers", mock.Anything, "bob", "").Return([]*domain.User{}, dbError)
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

			c, rec := newContext(http.MethodGet, "/users/search"+tt.query, "")

			err := SearchUser(svc)(c)

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
