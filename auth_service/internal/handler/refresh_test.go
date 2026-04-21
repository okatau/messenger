package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"auth_service/internal/domain"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_RefreshToken(t *testing.T) {
	userInfo := domain.AuthSession{AccessToken: "access_token"}
	tests := []struct {
		name       string
		body       string
		setup      func(s *mockAuthService)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: `{"refresh_token": "refresh_token"}`,
			setup: func(s *mockAuthService) {
				s.On("Refresh", mock.Anything, "refresh_token").Return(&userInfo, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid request body",
			body:       `{bad}`,
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid refresh token",
			body:       `{"refresh_token": ""}`,
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: `{"refresh_token": "refresh_token"}`,
			setup: func(s *mockAuthService) {
				s.On("Refresh", mock.Anything, "refresh_token").Return((*domain.AuthSession)(nil), errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{}
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
				assert.Contains(t, resp, "access_token")
			}
			svc.AssertExpectations(t)
		})
	}
}
