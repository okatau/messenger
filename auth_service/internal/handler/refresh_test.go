package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"auth_service/internal/domain"
	"auth_service/internal/handler/mocks"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_RefreshToken(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(s *mocks.MockAuth)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: `{"refreshToken": "refreshToken"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Refresh(mock.Anything, "refreshToken").Return(&domain.AuthSession{AccessToken: "access_token"}, nil)
			},
			wantStatus: http.StatusOK,
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
			name: "token expired",
			body: `{"refreshToken": "refreshToken"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Refresh(mock.Anything, "refreshToken").Return(nil, domain.ErrTokenExpired)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "token not found",
			body: `{"refreshToken": "refreshToken"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Refresh(mock.Anything, "refreshToken").Return(nil, domain.ErrTokenNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: `{"refreshToken": "refreshToken"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Refresh(mock.Anything, "refreshToken").Return((*domain.AuthSession)(nil), dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockAuth(t)
			tt.setup(svc)

			_, c, rec := newContext(http.MethodPost, "/refresh", tt.body)

			err := Refresh(svc)(c)

			if tt.wantErr {
				var echoError *echo.HTTPError
				require.ErrorAs(t, err, &echoError)
				assert.Equal(t, tt.wantStatus, echoError.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Contains(t, resp, "accessToken")
			}
		})
	}
}
