package handler

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"auth_service/internal/domain"
	"auth_service/internal/handler/mocks"
)

func Test_Logout(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(s *mocks.MockAuth)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: `{"refreshToken": "refresh_token"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Logout(mock.Anything, "refresh_token").Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid request body",
			body:       `{bad}`,
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid refresh token",
			body:       `{"refreshToken": ""}`,
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "token not found",
			body: `{"refreshToken": "refreshToken"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Logout(mock.Anything, "refreshToken").Return(domain.ErrTokenNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: `{"refreshToken": "refreshToken"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Logout(mock.Anything, "refreshToken").Return(dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockAuth(t)
			tt.setup(svc)

			_, c, rec := newContext(http.MethodPost, "/logout", tt.body)

			err := Logout(svc)(c)

			if tt.wantErr {
				var echoError *echo.HTTPError
				require.ErrorAs(t, err, &echoError)
				assert.Equal(t, tt.wantStatus, echoError.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}
		})
	}
}
