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

func Test_Register(t *testing.T) {
	getBody := func(username, email, password string) string {
		body, _ := json.Marshal(
			struct {
				Username string `json:"username"`
				Email    string `json:"email"`
				Password string `json:"password"`
			}{
				Email:    email,
				Username: username,
				Password: password,
			},
		)
		return string(body)
	}

	tests := []struct {
		name       string
		body       string
		setup      func(s *mocks.MockAuth)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: getBody(aliceName, aliceMail, alicePW),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Register(mock.Anything, aliceName, aliceMail, alicePW).Return(&domain.User{ID: "id"}, nil)
			},
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "invalid request body",
			body:       "{bad}",
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid email",
			body:       getBody(aliceName, "", alicePW),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid password",
			body:       getBody(aliceName, aliceMail, ""),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid username",
			body:       getBody("a", aliceMail, alicePW),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "user exists",
			body: getBody(aliceMail, aliceMail, alicePW),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Register(mock.Anything, aliceMail, aliceMail, alicePW).Return((*domain.User)(nil), domain.ErrUserExists)
			},
			wantStatus: http.StatusConflict,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: getBody(aliceMail, aliceMail, alicePW),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Register(mock.Anything, aliceMail, aliceMail, alicePW).Return((*domain.User)(nil), dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockAuth(t)
			tt.setup(svc)

			_, c, rec := newContext(http.MethodPost, "/register", tt.body)

			err := Register(svc)(c)

			if tt.wantErr {
				var echoError *echo.HTTPError
				require.ErrorAs(t, err, &echoError)
				assert.Equal(t, tt.wantStatus, echoError.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Contains(t, resp, "userId")
			}
		})
	}
}
