package handler

import (
	"auth_service/internal/handler/mocks"
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
		setup      func(s *mocks.MockAuth)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: `{"refresh_token": "refresh_token"}`,
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
			body:       `{"refresh_token": ""}`,
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "invalid internal server error",
			body: `{"refresh_token": "refresh_token"}`,
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Logout(mock.Anything, "refresh_token").Return(dbError)
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
