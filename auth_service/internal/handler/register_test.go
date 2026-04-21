package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auth_service/internal/domain"
	"auth_service/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAuthService struct{ mock.Mock }

func (m *mockAuthService) Register(ctx context.Context, name, email, password string) (*domain.User, error) {
	args := m.Called(ctx, name, email, password)
	log.Println(args)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*domain.AuthSession, error) {
	args := m.Called(ctx, email, password)
	return args.Get(0).(*domain.AuthSession), args.Error(1)
}

func (m *mockAuthService) Refresh(ctx context.Context, token string) (*domain.AuthSession, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(*domain.AuthSession), args.Error(1)
}

func (m *mockAuthService) Logout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

var _ service.Auth = (*mockAuthService)(nil)

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
		setup      func(s *mockAuthService)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: getBody(name, email, password),
			setup: func(s *mockAuthService) {
				s.On("Register", mock.Anything, name, email, password).Return(&user, nil)
			},
			wantStatus: http.StatusCreated,
			wantErr:    false,
		},
		{
			name:       "invalid request body",
			body:       "{bad}",
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid email",
			body:       getBody(name, "", password),
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid password",
			body:       getBody(name, email, ""),
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid username",
			body:       getBody("", email, password),
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: getBody(name, email, password),
			setup: func(s *mockAuthService) {
				s.On("Register", mock.Anything, name, email, password).Return((*domain.User)(nil), errors.New("db down"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{}
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
			svc.AssertExpectations(t)
		})
	}
}
