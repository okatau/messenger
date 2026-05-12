package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"auth_service/internal/domain"
	"auth_service/internal/handler/mocks"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Login(t *testing.T) {
	email := "email@mail.com"
	password := "password"

	AuthSession := domain.AuthSession{UserID: uuid.NewString(), Username: "user", RefreshToken: "refresh_token", AccessToken: "access_token"}
	getBody := func(email, password string) string {
		body, _ := json.Marshal(
			struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}{
				Email:    email,
				Password: password,
			})

		return string(body)
	}

	tests := []struct {
		name       string
		body       string
		setup      func(s *mocks.MockAuth)
		wantStatus int
		wantError  bool
	}{
		{
			name: "success",
			body: getBody(email, password),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Login(mock.Anything, email, password).Return(&AuthSession, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid request body",
			body:       "{bad}",
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid email",
			body:       getBody("", password),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid password",
			body:       getBody(email, ""),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "internal server error",
			body: getBody(email, password),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Login(mock.Anything, email, password).Return((*domain.AuthSession)(nil), dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockAuth(t)
			tt.setup(svc)

			_, c, rec := newContext(http.MethodPost, "/login", tt.body)

			err := Login(svc)(c)

			if tt.wantError {
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
		})
	}
}
