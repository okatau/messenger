package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Logout(t *testing.T) {
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
				s.On("Logout", mock.Anything, "refresh_token").Return(nil)
			},
			wantStatus: http.StatusNoContent,
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
			name: "invalid internal server error",
			body: `{"refresh_token": "refresh_token"}`,
			setup: func(s *mockAuthService) {
				s.On("Logout", mock.Anything, "refresh_token").Return(errors.New("db"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{}
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
			svc.AssertExpectations(t)
		})
	}
}
