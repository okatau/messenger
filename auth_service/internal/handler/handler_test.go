package handler

import (
	"auth_service/internal/domain"
	"errors"

	// "auth_service/internal/handler"
	"auth_service/internal/service"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*domain.UserInfo, error) {
	args := m.Called(ctx, email, password)
	return args.Get(0).(*domain.UserInfo), args.Error(1)
}

func (m *mockAuthService) Refresh(ctx context.Context, token string) (*domain.UserInfo, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(*domain.UserInfo), args.Error(1)
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
		Name:      name,
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

func Test_Login(t *testing.T) {
	email := "email@mail.com"
	password := "password"

	userInfo := domain.UserInfo{UserID: uuid.NewString(), Username: "user", RefreshToken: "refresh_token", AccessToken: "access_token"}
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
		setup      func(s *mockAuthService)
		wantStatus int
		wantError  bool
	}{
		{
			name: "success",
			body: getBody(email, password),
			setup: func(s *mockAuthService) {
				s.On("Login", mock.Anything, email, password).Return(&userInfo, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid request body",
			body:       "{bad}",
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid email",
			body:       getBody("", password),
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid password",
			body:       getBody(email, ""),
			setup:      func(s *mockAuthService) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "internal server error",
			body: getBody(email, password),
			setup: func(s *mockAuthService) {
				s.On("Login", mock.Anything, email, password).Return((*domain.UserInfo)(nil), errors.New("db down"))
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAuthService{}
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
			svc.AssertExpectations(t)
		})
	}
}

func Test_RefreshToken(t *testing.T) {
	userInfo := domain.UserInfo{AccessToken: "access_token"}
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
				s.On("Refresh", mock.Anything, "refresh_token").Return((*domain.UserInfo)(nil), errors.New("db"))
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
