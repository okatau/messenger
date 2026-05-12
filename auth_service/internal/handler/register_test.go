package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/handler/mocks"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var dbError = errors.New("db error")

func newContext(method, target, body string) (*echo.Echo, *echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	return e, e.NewContext(req, rec), rec
}

func Test_Register(t *testing.T) {
	email := "email@mail.com"
	name := "username"
	password := "password"

	user := domain.User{
		ID:        uuid.NewString(),
		Username:  name,
		Email:     email,
		CreatedAt: time.Now(),
	}

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
			body: getBody(name, email, password),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Register(mock.Anything, name, email, password).Return(&user, nil)
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
			body:       getBody(name, "", password),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid password",
			body:       getBody(name, email, ""),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid username",
			body:       getBody("", email, password),
			setup:      func(s *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: getBody(name, email, password),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Register(mock.Anything, name, email, password).Return((*domain.User)(nil), dbError)
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
				assert.Contains(t, resp, "user_id")
			}
		})
	}
}
